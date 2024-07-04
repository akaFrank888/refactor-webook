package limiter

import (
	"context"
	_ "embed"
	"github.com/redis/go-redis/v9"
	"time"
)

//go:embed slide_window.lua
var luaScript string

// RedisSlidingWindowLimiter 基于 Redis 实现的滑动窗口限流结构体
type RedisSlidingWindowLimiter struct {
	cmd      redis.Cmdable
	interval time.Duration
	// 阈值
	rate int
}

func NewRedisSlidingWindowLimiter(cmd redis.Cmdable, interval time.Duration, rate int) *RedisSlidingWindowLimiter {
	return &RedisSlidingWindowLimiter{cmd: cmd, interval: interval, rate: rate}
}

func (r *RedisSlidingWindowLimiter) Limit(ctx context.Context, key string) (bool, error) {
	return r.cmd.Eval(ctx, luaScript, []string{key},
		r.interval.Milliseconds(), r.rate, time.Now().UnixMilli()).Bool()
}
