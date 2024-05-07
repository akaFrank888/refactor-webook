package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"log"
	"net/http"
	"refactor-webook/webook/internal/web"
	"strings"
	"time"
)

type LoginJWTMiddleWareBuilder struct {
}

func NewLoginJWTMiddleWareBuilder() *LoginJWTMiddleWareBuilder {
	return &LoginJWTMiddleWareBuilder{}
}

func (b *LoginJWTMiddleWareBuilder) CheckLogin() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		path := ctx.Request.URL.Path
		if path == "/users/signup" ||
			path == "/users/login" ||
			path == "/users/login_sms/code/send" ||
			path == "/users/login_sms" {
			return
		}
		// 一、JTW的登录校验：解析JWT
		header := ctx.GetHeader("Authorization")
		if header == "" {
			ctx.AbortWithStatus(http.StatusUnauthorized)
		}
		if len(strings.Split(header, " ")) != 2 {
			// header不是Bear ** 形式
			ctx.AbortWithStatus(http.StatusUnauthorized)
		}

		tokenStr := strings.Split(header, " ")[1]
		uc := web.UserClaims{}
		// note 1. keyfunc的作用是生成更高级的JWTKey，但我们不需要对key设计func，用固定的即可。 2. &uc不是uc
		token, err := jwt.ParseWithClaims(tokenStr, &uc, func(token *jwt.Token) (interface{}, error) {
			return web.JWTKey, nil
		})
		if err != nil {
			// token解析不出来（可能是伪造的）
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if !token.Valid {
			// token解析出来了，但过期了（ uc.ExpireAt.Before(time.Now()) == True）
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if uc.UserAgent != ctx.GetHeader("user-agent") {
			// todo 埋点，正常用户不会进入该分支
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// 二、刷新JWT的过期时间
		expireTime := uc.ExpiresAt
		// 假设JWT的过期时间是1min，实现这次访问时距过期时间不到30s就刷新
		if expireTime.Sub(time.Now()) < time.Second*30 {
			uc.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Minute))
			tokenStr, err := token.SignedString(web.JWTKey)
			if err != nil {
				log.Println(err)
			} else {
				ctx.Header("x-jwt-token", tokenStr)
			}
		}

		// 三、保存uc到ctx中
		ctx.Set("user", uc)
	}
}
