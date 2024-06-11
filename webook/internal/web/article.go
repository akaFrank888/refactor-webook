package web

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"refactor-webook/webook/internal/domain"
	"refactor-webook/webook/internal/service"
	"refactor-webook/webook/internal/web/jwt"
	"refactor-webook/webook/pkg/kit"
	"refactor-webook/webook/pkg/logger"
	"time"
)

type ArticleHandler struct {
	svc service.ArticleService
	l   logger.LoggerV1
}

func NewArticleHandler(svc service.ArticleService, l logger.LoggerV1) *ArticleHandler {
	return &ArticleHandler{
		svc: svc,
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
	g.POST("/list", h.List)        // note offset和limit不通过get方式拼接在url中，而是直接post到后端

	// 读者的查询接口

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

func (h *ArticleHandler) Detail(ctx *gin.Context) {

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
