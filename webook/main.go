package main

import (
	"github.com/gin-contrib/sessions"
	redis_contrib "github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"net/http"
	"refactor-webook/webook/internal/web/middleware"
)

func main() {

	initLogger()
	app := InitWebServer()
	initPrometheus()
	for _, c := range app.consumers {
		err := c.Start()
		if err != nil {
			panic(err)
		}
	}
	server := app.server
	server.GET("/hello", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "hello，启动成功")
	})
	server.Run(":8080")

}
func initLogger() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(logger)
}

func initPrometheus() {
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":8081", nil)
	}()
}

func useJWT(server *gin.Engine) {
	login := middleware.LoginJWTMiddleWareBuilder{}
	server.Use(login.CheckLogin())
}

func useSession(server *gin.Engine) {
	login := middleware.LoginMiddleWareBuilder{}
	//// note session存储方式一：基于cookie存储
	//store := cookie.NewStore([]byte("secret"))

	//// note session存储方式二：基于memstore（内存）存储
	//// note 第一个是 authentication keys（验证来源和是否被篡改），第二个是 encryption keys（进行加密）
	//store = memstore.NewStore([]byte("CIft1b5qZjyLcc0zZo2UrUx5rk3KE0LvZKv73fw502oXd6vfYu1OAQvbSel8whvm"),
	//	[]byte("zfIxdNzQo55gAc1wZvhtlulPQ9eI4YbzyjtfNwHNxsY1SnZ7Bhd4Kd9xoBu23tTc"))

	// note session存储方式三：基于redis存储
	store, err := redis_contrib.NewStore(16, "tcp", "localhost:6379", "",
		// 不要写成64位，bug找了好久
		[]byte("0aPe1L0TQxjcBN9nPRxyDbhuBEnUUhDg"),
		[]byte("0aPe1L0TQxjcBN9nPRxyDbhuBEnUUhDg"))
	if err != nil {
		panic(err)
	}

	server.Use(sessions.Sessions("ssid", store)) // note 创建session返回的是 HandlerFunc类型
	// 实现登录校验：查看该session中有无userId
	server.Use(login.CheckLogin())
}
