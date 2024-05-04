package middleware

import (
	"encoding/gob"
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

type LoginMiddleWareBuilder struct {
}

func (b *LoginMiddleWareBuilder) CheckLogin() gin.HandlerFunc {
	// 注意gob.Register写的位置
	gob.Register(time.Now())
	return func(ctx *gin.Context) {
		if ctx.Request.URL.Path == "/users/signup" ||
			ctx.Request.URL.Path == "/users/login" {
			return
		}
		// 登录校验
		session := sessions.Default(ctx)
		userId := session.Get("userId")
		if userId == nil {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// 刷新session的过期时间
		const updateTimeKey = "update_time"
		now := time.Now()
		val := session.Get(updateTimeKey)
		lastUpdateTime, ok := val.(time.Time)
		// 或者 gob.Register(time.Time{})
		if !ok || now.Sub(lastUpdateTime) > time.Second*40 {
			// 1. 没取出上一次updateTime 2. 据上次更新时间超过10s   ===》  就再次更新
			// note time.Time类型不能直接set到redis中，需要先用gob注册一下（因为time.Time是go语言中的类型）
			session.Set(updateTimeKey, now)
			// note session在使用set时要重新set一边其他的key，因为会覆盖；记得save
			session.Set("userId", userId)
			err := session.Save()
			if err != nil {
				// 这个err级别不需要panic，只用打日志，因为只是没刷新登录状态而已
				fmt.Println(err)
			}
		}
	}
}
