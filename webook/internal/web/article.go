package web

import (
	"context"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
	"net/http"
	"refactor-webook/webook/internal/domain"
	"refactor-webook/webook/internal/service"
	"refactor-webook/webook/internal/web/jwt"
	"refactor-webook/webook/pkg/kit"
	"refactor-webook/webook/pkg/logger"
	"strconv"
	"time"
)

type ArticleHandler struct {
	svc      service.ArticleService
	interSvc service.InteractiveService

	biz string
	l   logger.LoggerV1
}

func NewArticleHandler(svc service.ArticleService, interSvc service.InteractiveService, l logger.LoggerV1) *ArticleHandler {
	return &ArticleHandler{
		svc:      svc,
		interSvc: interSvc,

		biz: "article",
		l:   l,
	}
}

func (h *ArticleHandler) RegisterRoutes(server *gin.Engine) {
	g := server.Group("/articles")
	g.POST("/edit", h.Edit)
	g.POST("/publish", h.Publish)
	g.POST("/withdraw", h.Withdraw)

	// 创作者的查询接口
	g.GET("/detail/:id", h.Detail) // 参数路由
	g.POST("/list", h.List)        // note offset和 limit不通过 get方式拼接在url中，而是直接 post到后端

	// 读者的查询接口 (查线上库)
	p := g.Group("/pub")
	p.GET("/:id", h.PubDetail)
	p.POST("/like", h.Like)       // 点赞或取消点赞
	p.POST("/collect", h.Collect) // 点赞或取消点赞

}

// Edit 约定：接收 Article 输入，返回 文章的ID
func (h *ArticleHandler) Edit(ctx *gin.Context) {
	type Req struct {
		Id      int64  `json:"id"`
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	var req Req
	if err := ctx.Bind(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, Result{
			Code: 5,
			Msg:  "系统错误",
		})
	}

	uc := ctx.MustGet("user").(jwt.UserClaims)
	id, err := h.svc.Save(ctx, domain.Article{
		Id:      req.Id,
		Title:   req.Title,
		Content: req.Content,
		Author: domain.Author{
			Id: uc.Uid,
		},
	})
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		// 打印日志
		h.l.Error("保存文章数据失败", logger.Error(err), logger.Int64("uid", uc.Uid))
	}
	ctx.JSON(http.StatusOK, Result{
		Data: id,
	})
}

// Publish note 根据是否传 Id 判断是否是已有帖子
func (h *ArticleHandler) Publish(ctx *gin.Context) {
	type Req struct {
		Id      int64  `json:"id"`
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	var req Req
	if err := ctx.Bind(&req); err != nil {
		return
	}

	uc := ctx.MustGet("user").(jwt.UserClaims)
	id, err := h.svc.Publish(ctx, domain.Article{
		Id:      req.Id,
		Title:   req.Title,
		Content: req.Content,
		Author: domain.Author{
			Id: uc.Uid,
		},
	})
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		// 打印日志
		h.l.Error("发表文章失败", logger.Error(err), logger.Int64("uid", uc.Uid))
	}
	ctx.JSON(http.StatusOK, Result{
		Data: id,
	})
}

func (h *ArticleHandler) Withdraw(ctx *gin.Context) {
	type Req struct {
		Id int64 `json:"id"`
	}
	var req Req
	if err := ctx.Bind(&req); err != nil {
		return
	}

	uc := ctx.MustGet("user").(jwt.UserClaims)
	err := h.svc.Withdraw(ctx, uc.Uid, req.Id)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		// 打印日志
		h.l.Error("撤回文章数据失败",
			logger.Error(err),
			logger.Int64("uid", uc.Uid),
			logger.Int64("aid", req.Id),
		)
		return
	}
	ctx.JSON(http.StatusOK, Result{
		Msg: "ok",
	})
}

func (h *ArticleHandler) List(ctx *gin.Context) {
	var page Page
	if err := ctx.Bind(&page); err != nil {
		return
	}

	uc := ctx.MustGet("user").(jwt.UserClaims)
	articles, err := h.svc.GetByAuthor(ctx, uc.Uid, page.Offset, page.Limit)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		h.l.Error("查找文章列表失败",
			logger.Error(err),
			logger.Int64("uid", uc.Uid),
			logger.Int("offset", page.Offset),
			logger.Int("limit", page.Limit),
		)
		return
	}

	ctx.JSON(http.StatusOK, Result{
		// note 不能直接把 domain 的数据暴露给前端
		Data: kit.Map[domain.Article, ArticleVo](articles, func(idx int, src domain.Article) ArticleVo {
			return ArticleVo{
				Id: src.Id,
				// 不需要返回article的content和AuthorId
				Title:    src.Title,
				Status:   src.Status.ToUint8(),
				AuthorId: src.Author.Id,

				Abstract: src.Abstract(),
				// note 对时间进行格式转换！
				Ctime: src.Ctime.Format(time.DateTime),
				Utime: src.Utime.Format(time.DateTime),
			}
		}),
	})
}

func (h *ArticleHandler) Detail(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 4,
			Msg:  "参数错误",
		})
		h.l.Error("id 非法格式",
			logger.String("id", idStr),
			logger.Error(err))
		return
	}
	article, err := h.svc.GetById(ctx, id)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		h.l.Error("通过id获取文章失败",
			logger.Int64("id", id),
			logger.Error(err))
		return
	}
	// note 取出article后要立即校验作者id
	uc := ctx.MustGet("user").(jwt.UserClaims)
	if article.Author.Id != uc.Uid {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		h.l.Error("非法访问article，作者id不匹配",
			logger.Int64("uid", uc.Uid), // 输出执行操作的用户id
			logger.Error(err))
		return
	}

	vo := ArticleVo{
		Id: article.Id,
		// 不需要返回article的abstract和AuthorId
		Title:   article.Title,
		Status:  article.Status.ToUint8(),
		Content: article.Content,
		// note 对时间进行格式转换！
		Ctime: article.Ctime.Format(time.DateTime),
		Utime: article.Utime.Format(time.DateTime),
	}

	ctx.JSON(http.StatusOK, Result{
		Data: vo,
	})

}

func (h *ArticleHandler) PubDetail(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 4,
			Msg:  "参数错误",
		})
		h.l.Error("id 非法格式",
			logger.String("id", idStr),
			logger.Error(err))
		return
	}

	// 在查看文章时返回对应的interactive
	uc := ctx.MustGet("user").(jwt.UserClaims)
	var (
		eg      errgroup.Group
		article domain.Article
		inter   domain.Interactive
	)
	// note 1. 开启errgroup 2. goroutine中不要复用外面的error
	eg.Go(func() error {
		var er error
		article, er = h.svc.GetPubById(ctx, id, uc.Uid)
		return er
	})
	eg.Go(func() error {
		var er error
		inter, er = h.interSvc.Get(ctx, h.biz, id, uc.Uid)
		return er
	})

	err = eg.Wait()
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		h.l.Error("通过id获取文章的interactive失败",
			logger.Int64("aid", id),
			logger.Int64("uid", uc.Uid),
			logger.Error(err))
		return
	}

	// note 看完一篇article后，文章（某资源）的阅读数+1，用 异步 实现
	go func() {
		// 1. 如果你想摆脱原本主链路的超时控制，你就创建一个新的
		// 2. 如果你不想，你就用 ctx
		newCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		er := h.interSvc.IncrReadCnt(newCtx, h.biz, id)
		if er != nil {
			h.l.Error("更新阅读数失败",
				logger.Int64("aid", id),
				logger.Error(err))
		}
	}()

	vo := ArticleVo{
		Id: article.Id,
		// note 相比于创作者的 Detail()，需要多返回创作者的 id 和 name [此name需要在repo层结合userRepo进行封装对象]
		Title:      article.Title,
		Status:     article.Status.ToUint8(),
		AuthorId:   article.Author.Id,
		AuthorName: article.Author.Name,
		Content:    article.Content,

		ReadCnt:    inter.ReadCnt,
		LikeCnt:    inter.LikeCnt,
		CollectCnt: inter.CollectCnt,
		Liked:      inter.Liked,
		Collected:  inter.Collected,

		Ctime: article.Ctime.Format(time.DateTime),
		Utime: article.Utime.Format(time.DateTime),
	}
	ctx.JSON(http.StatusOK, Result{
		Data: vo,
	})
}

func (h *ArticleHandler) Like(ctx *gin.Context) {
	type Req struct {
		Id int64 `json:"id"`
		// ture:点赞   false:取消点赞
		Like bool `json:"like"`
	}

	var req Req
	if err := ctx.Bind(&req); err != nil {
		return
	}
	uc := ctx.MustGet("user").(jwt.UserClaims)
	// note 将点赞和取消点赞视为两个业务逻辑，所以在web层进行分发
	var err error
	if req.Like {
		err = h.interSvc.Like(ctx, h.biz, req.Id, uc.Uid)
	} else {
		err = h.interSvc.CancelLike(ctx, h.biz, req.Id, uc.Uid)
	}

	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Msg: "系统错误",
		})
		h.l.Error("点赞/取消点赞失败",
			logger.Error(err),
			logger.Int64("uid", uc.Uid),
			logger.Int64("aid", req.Id))
		return
	}
	ctx.JSON(http.StatusOK, Result{
		Msg: "ok",
	})

}

func (h *ArticleHandler) Collect(ctx *gin.Context) {
	type Req struct {
		Id int64 `json:"id"`
		// ture:收藏   false:取消收藏
		Collect bool `json:"collect"`
		// 收藏夹的id
		Cid int64 `json:"cid"`
	}

	var req Req
	if err := ctx.Bind(&req); err != nil {
		return
	}
	uc := ctx.MustGet("user").(jwt.UserClaims)
	// note 将点赞和取消点赞视为两个业务逻辑，所以在web层进行分发
	var err error
	if req.Collect {
		err = h.interSvc.Collect(ctx, h.biz, req.Id, uc.Uid, req.Cid)
	} else {
		err = h.interSvc.CancelCollect(ctx, h.biz, req.Id, uc.Uid, req.Cid)
	}

	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Msg: "系统错误",
		})
		h.l.Error("收藏/取消收藏失败",
			logger.Error(err),
			logger.Int64("uid", uc.Uid),
			logger.Int64("aid", req.Id))
		return
	}
	ctx.JSON(http.StatusOK, Result{
		Msg: "ok",
	})
}
