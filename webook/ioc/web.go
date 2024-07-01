package ioc

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"refactor-webook/webook/internal/web"
	ijwt "refactor-webook/webook/internal/web/jwt"
	"refactor-webook/webook/internal/web/middleware"
	"refactor-webook/webook/pkg/ginx/prometheus"
	"refactor-webook/webook/pkg/logger"
	"strings"
	"time"
)

func InitWebServer(funcs []gin.HandlerFunc,
	userHdl *web.UserHandler, wechatHdl *web.OAuth2WechatHandler, articleHdl *web.ArticleHandler) *gin.Engine {
	server := gin.Default()
	server.Use(funcs...)
	// 注册路由
	userHdl.RegisterRoutes(server)
	wechatHdl.RegisterRoutes(server)
	articleHdl.RegisterRoutes(server)
	return server
}

func InitGinMiddlewares(hdl ijwt.Handler, l logger.LoggerV1) []gin.HandlerFunc {
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

		//// note 限流
		//ratelimit.NewBuilder(redis.NewClient(&redis.Options{
		//	Addr: "localhost:6379",
		//}), time.Second, 100).Build(),
		//
		//// note 日志
		//accesslog.NewLogMiddlewareBuilder(func(ctx context.Context, al accesslog.AccessLog) {
		//	// 打印 debug 级别的
		//	l.Debug("", logger.Field{Key: "req", Val: al})
		//}).AllowReqBody().AllowRespBody().Build(),

		// note prometheus
		prometheus.NewBuilder(
			"ecommerce_BG", "webook", "gin_http", "", "统计 gin 的http接口数据",
		).BuildResponseTime(),
		prometheus.NewBuilder(
			"ecommerce_BG", "webook", "gin_http", "", "统计 gin 的http接口数据",
		).BuildActiveRequest(),

		// JWT
		middleware.NewLoginJWTMiddleWareBuilder(hdl).CheckLogin(),
	}

}
