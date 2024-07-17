package ioc

import (
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	tencentSMS "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20210111"
	"os"
	"refactor-webook/webook/internal/service/localsms"
	"refactor-webook/webook/internal/sms_project/service"
	"refactor-webook/webook/internal/sms_project/service/tencent"
)

func InitSMSService() service.Service {

	//return ratelimit.NewSmsServiceRateLimit(initTencentSMSService(), limiter.NewRedisSlidingWindowLimiter(
	//	redis.NewClient(&redis.Options{
	//		Addr: "localhost:6379",
	//	}), time.Second, 100))

	//return auth.NewSmsService(initTencentSMSService())
	return localsms.NewService()
}

// 腾讯的配置
func initTencentSMSService() service.Service {
	secretId, ok := os.LookupEnv("SMS_SECRET_ID")
	if !ok {
		panic("找不到腾讯 SMS 的 secret id")
	}
	secretKey, ok := os.LookupEnv("SMS_SECRET_KEY")
	if !ok {
		panic("找不到腾讯 SMS 的 secret key")
	}
	c, err := tencentSMS.NewClient(
		common.NewCredential(secretId, secretKey),
		"ap-nanjing",
		profile.NewClientProfile(),
	)
	if err != nil {
		panic(err)
	}
	return tencent.NewService(c, "1400842696", "妙影科技")
}

// 阿里的配置
