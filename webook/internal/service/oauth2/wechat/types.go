package wechat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"refactor-webook/webook/internal/domain"
	"refactor-webook/webook/pkg/logger"
)

type Service interface {
	OAuth2URL(ctx context.Context, state string) (string, error)
	VerifyCode(ctx context.Context, code string) (domain.WechatInfo, error)
}

type service struct {
	AppId     string
	AppSecret string
	// 发送http请求的client
	client *http.Client
	l      logger.LoggerV1
}

func NewService(appId, appSecret string, l logger.LoggerV1) Service {
	return &service{
		AppId:     appId,
		AppSecret: appSecret,
		client:    http.DefaultClient,
		l:         l,
	}
}

// 替换掉 APPID  REDIRECT_URI SCOPE STATE
const authURLPattern = `https://open.weixin.qq.com/connect/qrconnect?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_login&state=%s#wechat_redirect`
const redirectURL = `your_redirect_uri`

// 替换掉 APPID APP_SECRET CODE
const accessTokenURLPattern = `https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code`

func (s *service) OAuth2URL(ctx context.Context, state string) (string, error) {
	// 要对回调地址进行URL编码
	redirectUrl := url.PathEscape(redirectURL)
	// note state：用于标记像微信发送的url和微信返回的url【防止跨站请求伪造攻击】
	return fmt.Sprintf(authURLPattern, s.AppId, redirectUrl, state), nil
}

func (s *service) VerifyCode(ctx context.Context, code string) (domain.WechatInfo, error) {
	accessTokenURL := fmt.Sprintf(accessTokenURLPattern, s.AppId, s.AppSecret, code)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, accessTokenURL, nil)
	if err != nil {
		return domain.WechatInfo{}, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return domain.WechatInfo{}, err
	}

	var res Result
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return domain.WechatInfo{}, err
	}
	if res.ErrCode != 0 {
		return domain.WechatInfo{}, fmt.Errorf("调用微信接口失败 errcode %d, errmsg %s", res.ErrCode, res.ErrMsg)
	}

	return domain.WechatInfo{
		// todo 不需要返回AccessToken，只需要返回这两个id
		OpenId:  res.OpenId,
		UnionId: res.UnionId,
	}, nil

}

type Result struct {
	AccessToken string `json:"access_token"`
	// access_token接口调用凭证超时时间，单位（秒）
	ExpiresIn int64 `json:"expires_in"`
	// 用户刷新access_token
	RefreshToken string `json:"refresh_token"`
	// 授权用户唯一标识
	OpenId string `json:"openid"`
	// 用户授权的作用域，使用逗号（,）分隔
	Scope string `json:"scope"`
	// 当且仅当该网站应用已获得该用户的userinfo授权时，才会出现该字段。
	UnionId string `json:"unionid"`

	// 错误返回
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}
