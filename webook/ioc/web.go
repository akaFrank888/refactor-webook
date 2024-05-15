package ioc

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"refactor-webook/webook/internal/web"
	ijwt "refactor-webook/webook/internal/web/jwt"
	"refactor-webook/webook/internal/web/middleware"
	"strings"
	"time"
)

func InitWebServer(funcs []gin.HandlerFunc, userHdl *web.UserHandler, wechatHdl *web.OAuth2WechatHandler) *gin.Engine {
	server := gin.Default()
	server.Use(funcs...)
	// 注册路由
	userHdl.RegisterRoutes(server)
	wechatHdl.RegisterRoutes(server)
	return server
}

func InitGinMiddlewares(hdl ijwt.Handler) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		cors.New(cors.Config{
			AllowOrigins: []string{"http://localhost:3000"},
			// AllowMethods:     []string{"PUT", "PATCH"},
			// note JWT的跨域设置：AllowHeaders 和 ExposeHeaders
			// note authorization中的“Bear ***”
			AllowHeaders: []string{"authorization", "content-type"},
			// note 允许前端访问后端的响应中自定义的header
			ExposeHeaders:    []string{"x-jwt-token", "x-refresh-token"},
			AllowCredentials: true,
			AllowOriginFunc: func(origin string) bool {
				if strings.HasPrefix(origin, "http://localhost") {
					return true
				}
				return strings.Contains(origin, "your_company.com")
			},
			MaxAge: 12 * time.Hour,
		}),
		func(ctx *gin.Context) {
			println("第一个middleware")
		},
		func(ctx *gin.Context) {
			println("第二个middleware")
		},
		// JWT
		middleware.NewLoginJWTMiddleWareBuilder(hdl).CheckLogin(),
	}

}
