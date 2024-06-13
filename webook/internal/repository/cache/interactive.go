package cache

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
)

//go:embed lua/incr_cnt.lua
var luaIncrCnt string

const fieldReadCnt = "read_cnt"

type InteractiveCache interface {
	IncrReadCntIfExist(ctx context.Context, biz string, bizId int64) error
}

type RedisInteractiveCache struct {
	client     redis.Cmdable
	expiration time.Duration
}

func NewRedisInteractiveCache(client redis.Cmdable) InteractiveCache {
	return &RedisInteractiveCache{client: client, expiration: time.Minute * 15}
}

func (c *RedisInteractiveCache) IncrReadCntIfExist(ctx context.Context, biz string, bizId int64) error {
	key := c.key(biz, bizId)
	// 业务上：返回的1或0可以不考虑
	_, err := c.client.Eval(ctx, luaIncrCnt, []string{key}, fieldReadCnt, 1).Int()
	return err

}
func (c *RedisInteractiveCache) key(biz string, bizId int64) string {
	return fmt.Sprintf("interactive:%s:%d", biz, bizId)
}
