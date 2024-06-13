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
	ijwt "refactor-webook/webook/internal/web/jwt"
	"refactor-webook/webook/ioc"
)

var interactiveSvcSet = wire.NewSet(
	dao.NewGormInteractiveDao,
	cache.NewRedisInteractiveCache,
	repository.NewCachedInteractiveRepository,
	service.NewInteractiveService,
)

func InitWebServer() *gin.Engine {
	wire.Build(
		// 第三方依赖
		ioc.InitDB, ioc.InitRedis,
		ioc.InitLogger,
		// dao和cache
		dao.NewUserDao, cache.NewUserCache, cache.NewCodeCache,
		dao.NewArticleDao, cache.NewArticleCache,
		// repo
		repository.NewUserRepository, repository.NewCodeRepository,
		repository.NewArticleRepository,
		// service
		ioc.InitSMSService, service.NewUserService, service.NewCodeService,
		ioc.InitWechatService,
		service.NewArticleService,

		// handler
		web.NewUserHandler, web.NewOAuth2WechatHandler,
		ijwt.NewRedisJWTHandler,
		web.NewArticleHandler,

		// gin的中间件
		ioc.InitGinMiddlewares,
		// web 服务器
		ioc.InitWebServer,

		interactiveSvcSet,
	)
	return gin.Default()
}
