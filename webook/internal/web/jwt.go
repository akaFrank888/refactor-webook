package web

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"time"
)

type jwtHandler struct {
}

func (h *jwtHandler) setJWTToken(ctx *gin.Context, uid int64) {
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
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
	}
	// 添加进响应的header中
	ctx.Header("x-jwt-token", tokenStr)
}

var JWTKey = []byte("oIft1b5qZjyLcc0zZo2UrUx5rk3KE0LvZKv73fw502oXd6vfYu1OAQvbSel8whvm")

type UserClaims struct {
	jwt.RegisteredClaims
	Uid int64
	// note 利用请求头的User-Agent来增强安全性（防止jwt被攻击者获取）  User-Agent含有浏览器的信息
	UserAgent string
}
