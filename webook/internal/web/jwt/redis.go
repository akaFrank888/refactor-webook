package jwt

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"strings"
	"time"
)

type RedisJWTHandler struct {
	client redis.Cmdable
	// 长token的过期时间
	rcExpiration time.Duration
}

func NewRedisJWTHandler(client redis.Cmdable) Handler {
	return &RedisJWTHandler{
		client:       client,
		rcExpiration: 7 * 24 * time.Hour,
	}
}

// ExtractToken note 从 header 中的 Authorization 中 提取形如 “Bear **”的 token
func (h *RedisJWTHandler) ExtractToken(ctx *gin.Context) string {
	header := ctx.GetHeader("Authorization")
	if header == "" {
		// note 此处不需要处理，因为在后续解析 token 时会报错的
		return ""
	}
	if len(strings.Split(header, " ")) != 2 {
		// header不是Bear ** 形式
		return ""
	}
	tokenStr := strings.Split(header, " ")[1]
	return tokenStr
}

func (h *RedisJWTHandler) CheckSsid(ctx *gin.Context, ssid string) error {
	cnt, err := h.client.Exists(ctx, fmt.Sprintf("user:ssid:%s", ssid)).Result()
	// note 这种写法过于生硬，因为若redis崩了，正常登录着的用户也会无法通过这个判断导致返回401【可不处理err来降级处理，以兼容redis异常的情况】
	// note 要保证尽量提供服务，即使是有损的服务也比没服务好
	if err != nil {
		return err
	}
	if cnt > 0 {
		return errors.New("该用户已退出")
	}
	return nil
}

func (h *RedisJWTHandler) SetJWTToken(ctx *gin.Context, uid int64, ssid string) error {
	uc := UserClaims{
		Uid:       uid,
		Ssid:      ssid,
		UserAgent: ctx.GetHeader("user-agent"),
		// 定义JWT过期时间 —— 1min
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
		},
	}
	// 此token只是jwt的一个token结构体
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, uc)
	// 此tokenStr才是传输的token
	tokenStr, err := token.SignedString(JWTKey)
	if err != nil {
		return err
	}
	// 添加进响应的header中
	ctx.Header("x-jwt-token", tokenStr)
	return nil
}

// SetLoginToken note 在登录成功后，设置长短token和用于退出登录的ssid
func (h *RedisJWTHandler) SetLoginToken(ctx *gin.Context, uid int64) error {
	ssid := uuid.New().String()

	err := h.setRefreshToken(ctx, uid, ssid)
	if err != nil {
		return err
	}
	return h.SetJWTToken(ctx, uid, ssid)
}

func (h *RedisJWTHandler) ClearToken(ctx *gin.Context) error {
	// 1. 给前端非法的token
	ctx.Header("x-jwt-token", "")
	ctx.Header("x-refresh-token", "")
	// 2. 在redis写入ssid
	uc := ctx.MustGet("user").(UserClaims)
	// note 将ssid的过期时间设置为长token的过期时间
	return h.client.Set(ctx, fmt.Sprintf("user:ssid:%s", uc.Ssid), "", h.rcExpiration).Err()
}

func (h *RedisJWTHandler) setRefreshToken(ctx *gin.Context, uid int64, ssid string) error {
	rc := RefreshClaims{
		Uid:  uid,
		Ssid: ssid,
		RegisteredClaims: jwt.RegisteredClaims{
			// 长token过期时间为7天
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(h.rcExpiration)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, rc)
	tokenStr, err := token.SignedString(RefreshKey)
	if err != nil {
		return err
	}
	ctx.Header("x-refresh-token", tokenStr)
	return nil
}

var JWTKey = []byte("oIft1b5qZjyLcc0zZo2UrUx5rk3KE0LvZKv73fw502oXd6vfYu1OAQvbSel8whvm")
var RefreshKey = []byte("oIft1b5qZjyLcc0zZo2UrUx5r80iz0LvZKv73fw502oXd6vfYu1OAQvbSel8whvm")

type RefreshClaims struct {
	jwt.RegisteredClaims
	Uid  int64
	Ssid string
}

type UserClaims struct {
	jwt.RegisteredClaims
	Uid int64
	// note 利用请求头的User-Agent来增强安全性（防止jwt被攻击者获取）  User-Agent含有浏览器的信息
	UserAgent string
	Ssid      string
}
