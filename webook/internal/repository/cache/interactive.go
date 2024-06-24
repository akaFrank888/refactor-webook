package cache

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/redis/go-redis/v9"
	"refactor-webook/webook/internal/domain"
	"strconv"
	"time"
)

//go:embed lua/incr_cnt.lua
var luaIncrCnt string

const fieldReadCnt = "read_cnt"
const fieldLikeCnt = "like_cnt"
const fieldCollectionCnt = "collect_cnt"

type InteractiveCache interface {
	IncrReadCntIfExist(ctx context.Context, biz string, bizId int64) error
	IncrLikeCntIfExist(ctx context.Context, biz string, bizId int64) error
	DecrLikeCntIfExist(ctx context.Context, biz string, bizId int64) error
	IncrCollectCntIfExist(ctx context.Context, biz string, bizId int64) error
	DecrCollectCntIfExist(ctx context.Context, biz string, bizId int64) error
	Get(ctx context.Context, biz string, bizId int64) (domain.Interactive, error)
	Set(ctx context.Context, biz string, bizId int64, inter domain.Interactive) error
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

func (c *RedisInteractiveCache) IncrLikeCntIfExist(ctx context.Context, biz string, bizId int64) error {
	key := c.key(biz, bizId)
	_, err := c.client.Eval(ctx, luaIncrCnt, []string{key}, fieldLikeCnt, 1).Int()
	return err
}

func (c *RedisInteractiveCache) DecrLikeCntIfExist(ctx context.Context, biz string, bizId int64) error {
	key := c.key(biz, bizId)
	_, err := c.client.Eval(ctx, luaIncrCnt, []string{key}, fieldLikeCnt, -1).Int()
	return err
}

func (c *RedisInteractiveCache) IncrCollectCntIfExist(ctx context.Context, biz string, bizId int64) error {
	key := c.key(biz, bizId)
	_, err := c.client.Eval(ctx, luaIncrCnt, []string{key}, fieldCollectionCnt, 1).Int()
	return err
}

func (c *RedisInteractiveCache) DecrCollectCntIfExist(ctx context.Context, biz string, bizId int64) error {
	key := c.key(biz, bizId)
	_, err := c.client.Eval(ctx, luaIncrCnt, []string{key}, fieldCollectionCnt, -1).Int()
	return err
}

func (c *RedisInteractiveCache) Get(ctx context.Context, biz string, bizId int64) (domain.Interactive, error) {
	key := c.key(biz, bizId)
	res, err := c.client.HGetAll(ctx, key).Result() // note 返回的是一个 map[string]string
	if err != nil {
		return domain.Interactive{}, err
	}
	if len(res) == 0 {
		return domain.Interactive{}, ErrKeyNotExist
	}
	// 将取出来的string转成int，此处直接忽略掉错误
	var inter domain.Interactive
	inter.ReadCnt, _ = strconv.ParseInt(res[fieldReadCnt], 10, 64)
	inter.LikeCnt, _ = strconv.ParseInt(res[fieldLikeCnt], 10, 64)
	inter.CollectCnt, _ = strconv.ParseInt(res[fieldCollectionCnt], 10, 64)

	return inter, nil
}

func (c *RedisInteractiveCache) Set(ctx context.Context, biz string, bizId int64, inter domain.Interactive) error {
	key := c.key(biz, bizId)
	// note HSet()用于设置哈希表值的多个字段
	err := c.client.HSet(ctx, key,
		fieldReadCnt, inter.ReadCnt,
		fieldLikeCnt, inter.LikeCnt,
		fieldCollectionCnt, inter.CollectCnt,
	).Err()
	// 重新设置过期时间
	c.expiration = time.Minute * 15

	return err
}

func (c *RedisInteractiveCache) key(biz string, bizId int64) string {
	return fmt.Sprintf("interactive:%s:%d", biz, bizId)
}
