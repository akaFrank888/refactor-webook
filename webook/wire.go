//go:build wireinject

package main

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"refactor-webook/webook/internal/repository"
	"refactor-webook/webook/internal/repository/cache"
	"refactor-webook/webook/internal/repository/dao"
	"refactor-webook/webook/internal/service"
	"refactor-webook/webook/internal/web"
	"refactor-webook/webook/ioc"
)

func InitWebServer() *gin.Engine {
	wire.Build(
		// 第三方依赖
		ioc.InitDB, ioc.InitRedis, ioc.InitSMSService,
		// dao和cache
		dao.NewUserDao, cache.NewUserCache, cache.NewCodeCache,
		// repo
		repository.NewUserRepository, repository.NewCodeRepository,
		// service
		service.NewUserService, service.NewCodeService,
		// handler
		web.NewUserHandler,

		// gin的中间件
		ioc.InitGinMiddlewares,
		// web 服务器
		ioc.InitWebServer,
	)
	return gin.Default()
}
