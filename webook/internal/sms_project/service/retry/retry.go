package retry

import (
	"context"
	"errors"
	"refactor-webook/webook/internal/service/sms"
	"sync/atomic"
	"time"
)

type RetrySmsService struct {
	svc         sms.Service
	MaxAttempts int32
	Delay       time.Duration
}

func NewRetrySmsService(svc sms.Service, maxAttempts int32) *RetrySmsService {
	return &RetrySmsService{svc: svc, MaxAttempts: maxAttempts, Delay: time.Second * 3}
}

func (r *RetrySmsService) Send(ctx context.Context, tplId string, args []string, numbers ...string) error {
	atomic.StoreInt32(&r.MaxAttempts, 0)
	cnt := atomic.LoadInt32(&r.MaxAttempts)
	for cnt <= r.MaxAttempts {
		err := r.svc.Send(ctx, tplId, args, numbers...)
		switch err {
		case nil:
			atomic.StoreInt32(&r.MaxAttempts, 0)
			return nil
		case context.Canceled, context.DeadlineExceeded:
			return err
		default:
			// 其他错误，需要重试
			atomic.AddInt32(&r.MaxAttempts, 1)
		}
		time.Sleep(r.Delay)
	}

	// 重试失败的处理，可以是更换为下一个服务商等措施
	return errors.New("重试失败。。。")

}
