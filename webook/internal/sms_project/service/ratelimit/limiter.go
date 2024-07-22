package ratelimit

import (
	"context"
	"errors"
	"refactor-webook/webook/internal/sms_project/service"
	"refactor-webook/webook/pkg/limiter"
)

var ErrLimited = errors.New("触发短信服务限流")

type RateLimitSmsService struct {
	// svc是被装饰者
	svc service.Service
	l   limiter.Limiter
	key string
}

func (s *RateLimitSmsService) Send(ctx context.Context, tplId string, args []string, numbers ...string) error {
	// note 先执行起装饰作用的方法
	isLimit, err := s.l.Limit(ctx, s.key)
	if err != nil {
		return err
	}
	if isLimit {
		return ErrLimited
	}
	// note 最终委托被修饰的 svc 调用Send方法
	return s.svc.Send(ctx, tplId, args, numbers...)
}

func NewSmsServiceRateLimit(svc service.Service, l limiter.Limiter) *RateLimitSmsService {
	return &RateLimitSmsService{
		svc: svc,
		l:   l,
		key: "sms_project-tencent-limit",
	}
}
