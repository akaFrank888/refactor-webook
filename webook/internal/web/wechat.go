package web

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	uuid "github.com/lithammer/shortuuid/v4"
	"net/http"
	"refactor-webook/webook/internal/service"
	"refactor-webook/webook/internal/service/oauth2/wechat"
	ijwt "refactor-webook/webook/internal/web/jwt"
)

type OAuth2WechatHandler struct {
	ijwt.Handler
	svc     wechat.Service
	userSvc service.UserService

	stateCookieName string
}

func NewOAuth2WechatHandler(svc wechat.Service, userSvc service.UserService, hdl ijwt.Handler) *OAuth2WechatHandler {
	return &OAuth2WechatHandler{
		svc:             svc,
		userSvc:         userSvc,
		stateCookieName: "jwt-state",

		Handler: hdl,
	}
}

func (h *OAuth2WechatHandler) RegisterRoutes(server *gin.Engine) {
	g := server.Group("/oauth2/wechat")
	g.GET("/authurl", h.OAuth2URL)
	g.Any("/callback", h.Callback)
}

func (h *OAuth2WechatHandler) OAuth2URL(ctx *gin.Context) {
	state := uuid.New()
	url, err := h.svc.OAuth2URL(ctx, state)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "构造跳转wx登录的url失败",
		})
		return
	}
	err = h.setStateCookie(ctx, state)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "构造跳转wx登录的url中的state失败",
		})
	}
	ctx.JSON(http.StatusOK, Result{
		Data: url,
	})

}

func (h *OAuth2WechatHandler) Callback(ctx *gin.Context) {

	// 先检查state是否一致
	err := h.verifyState(ctx)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "state验证失败",
		})
		return
	}

	code := ctx.Query("code")
	wechatInfo, err := h.svc.VerifyCode(ctx, code)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "wechat的code失败",
		})
		return
	}

	// 拿wechatInfo中的openId判断是否已经注册或直接登录
	u, err := h.userSvc.FindOrCreateByWechat(ctx, wechatInfo)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "wechat注册或登录失败",
		})
	}
	err = h.SetLoginToken(ctx, u.Id)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "设置jwt失败",
		})
	}

	ctx.JSON(http.StatusOK, Result{
		Msg: "登录成功",
	})
}

// note 将state存储在jwt中，再将jwt存cookie中【为什么存cookie而不存header？因为从wx返回来的时候是直接到后端的，如果经过前端，则前端就可以加入header了】
func (h *OAuth2WechatHandler) setStateCookie(ctx *gin.Context, state string) error {
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, StateClaims{
		State: state,
	})
	tokenStr, err := token.SignedString(ijwt.JWTKey)
	if err != nil {
		return err
	}
	// 放入cookie中
	// note 线上环境的secure参数再改为true
	ctx.SetCookie(h.stateCookieName, tokenStr, 600, "/oauth2/wechat/callback", "", false, true)
	return nil
}

// note 将cookie中的state和url中的state对比
func (h *OAuth2WechatHandler) verifyState(ctx *gin.Context) error {
	state := ctx.Query("state")
	tokenStr, err := ctx.Cookie(h.stateCookieName)
	if err != nil {
		return err
	}
	var sc StateClaims
	_, err = jwt.ParseWithClaims(tokenStr, &sc, func(token *jwt.Token) (any, error) {
		return ijwt.JWTKey, nil
	})
	if err != nil {
		return errors.New("cookie不是合法的jwt token")
	}
	if sc.State != state {
		return errors.New("state被篡改了")
	}
	return nil
}

type StateClaims struct {
	jwt.RegisteredClaims
	State string
}
