package web

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"strings"
	"time"
)

type jwtHandler struct {
}

func (h *jwtHandler) setJWTToken(ctx *gin.Context, uid int64) error {

	err := h.setRefreshToken(ctx, uid)
	if err != nil {
		return err
	}

	uc := UserClaims{
		Uid:       uid,
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

func (h *jwtHandler) setRefreshToken(ctx *gin.Context, uid int64) error {
	rc := RefreshClaims{
		Uid: uid,
		RegisteredClaims: jwt.RegisteredClaims{
			// 长token过期时间为7天
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
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

// ExtractToken note 从 header 中的 Authorization 中 提取形如 “Bear **”的 token
func ExtractToken(ctx *gin.Context) string {
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

var JWTKey = []byte("oIft1b5qZjyLcc0zZo2UrUx5rk3KE0LvZKv73fw502oXd6vfYu1OAQvbSel8whvm")
var RefreshKey = []byte("oIft1b5qZjyLcc0zZo2UrUx5r80iz0LvZKv73fw502oXd6vfYu1OAQvbSel8whvm")

type RefreshClaims struct {
	jwt.RegisteredClaims
	Uid int64
}

type UserClaims struct {
	jwt.RegisteredClaims
	Uid int64
	// note 利用请求头的User-Agent来增强安全性（防止jwt被攻击者获取）  User-Agent含有浏览器的信息
	UserAgent string
}
