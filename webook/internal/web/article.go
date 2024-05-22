package web

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"refactor-webook/webook/internal/domain"
	"refactor-webook/webook/internal/service"
	"refactor-webook/webook/internal/web/jwt"
	"refactor-webook/webook/pkg/logger"
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
