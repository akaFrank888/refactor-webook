package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
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
		AllowHeaders:     []string{"authorization", "content-type"},
		ExposeHeaders:    []string{"Content-Length"},
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

	login := &middleware.LoginMiddleWareBuilder{}
	// note 创建基于cookie存储ssid的session会话
	store := cookie.NewStore([]byte("secret"))
	server.Use(sessions.Sessions("ssid", store)) // note 创建session返回的是 HandlerFunc类型
	// 实现登录校验：查看该session中有无userId
	server.Use(login.CheckLogin())

	return server
}

func initUserHdl(db *gorm.DB, server *gin.Engine) {
	userDao := dao.NewUserDao(db)
	userRepository := repository.NewUserRepository(userDao)
	userService := service.NewUserService(userRepository)
	h := web.NewUserHandler(userService)
	h.RegisterRoutes(server)
}
