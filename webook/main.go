package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/redis"
	"gorm.io/gorm"
	"refactor-webook/webook/internal/repository"
	"refactor-webook/webook/internal/repository/dao"
	"refactor-webook/webook/internal/service"
	"refactor-webook/webook/internal/web"
	"refactor-webook/webook/internal/web/middleware"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	server := initWebServer()

	db := dao.InitDB()
	initUserHdl(db, server)

	server.Run(":8080") // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}

func initWebServer() *gin.Engine {
	server := gin.Default()

	// note middleware的本质是 HandlerFunc
	server.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:3000"},
		// AllowMethods:     []string{"PUT", "PATCH"},
		// note JWT的跨域设置：AllowHeaders 和 ExposeHeaders
		// note authorization中的“Bear ***”
		AllowHeaders: []string{"authorization", "content-type"},
		// note 允许前端访问后端的响应中自定义的header
		ExposeHeaders:    []string{"x-jwt-token"},
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
		})

	useJWT(server) // note 被代替的方法：useSession(server)

	return server
}

func initUserHdl(db *gorm.DB, server *gin.Engine) {
	userDao := dao.NewUserDao(db)
	userRepository := repository.NewUserRepository(userDao)
	userService := service.NewUserService(userRepository)
	h := web.NewUserHandler(userService)
	h.RegisterRoutes(server)
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
	store, err := redis.NewStore(16, "tcp", "localhost:6379", "",
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
