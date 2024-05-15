package tencent

import (
	"context"
	"fmt"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tencentSMS "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20210111" // 引入sms
	"go.uber.org/zap"
	"refactor-webook/webook/internal/service/sms"
)

type Service struct {
	client *tencentSMS.Client
	// 腾讯云的短信SDK设计的就是string的指针
	appId    *string
	signName *string
}

func NewService(client *tencentSMS.Client, appId, signName string) sms.Service {
	return &Service{
		client:   client,
		appId:    &appId,
		signName: &signName,
	}

}

func (s *Service) Send(ctx context.Context, tplId string, args []string, numbers ...string) error {
	request := tencentSMS.NewSendSmsRequest()
	request.SmsSdkAppId = s.appId
	request.SignName = s.signName
	request.TemplateId = &tplId
	request.TemplateParamSet = common.StringPtrs(args)
	request.PhoneNumberSet = common.StringPtrs(numbers)
	response, err := s.client.SendSms(request)

	// note 研发环境经常用 debug 日志，线上环境不用
	zap.L().Debug("请求腾讯的SendSMS接口",
		zap.Any("request", request),
		zap.Any("response", response))

	if err != nil {
		fmt.Printf("An API error has returned: %s", err)
		return err
	}
	for _, statusPtr := range response.Response.SendStatusSet {
		// 解引用
		status := *statusPtr
		if status.Code != nil && *status.Code != "Ok" {
			return fmt.Errorf("短信发送失败 code:%s mag: %s", *status.Code, status.Message)
		}
	}
	return nil
}
