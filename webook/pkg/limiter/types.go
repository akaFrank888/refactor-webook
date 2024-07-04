package limiter

import "context"

type Limiter interface {
	// Limit key用来表示什么供应商的短信服务，如 'sms_tencent'
	Limit(ctx context.Context, key string) (bool, error)
}
