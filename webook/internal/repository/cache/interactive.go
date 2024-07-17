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

//go:embed lua/interactive_ranking_incr.lua
var luaRankingCnt string

//go:embed lua/interactive_ranking_set.lua
var luaRankingSet string

const fieldReadCnt = "read_cnt"
const fieldLikeCnt = "like_cnt"
const fieldCollectionCnt = "collect_cnt"

type InteractiveCache interface {
	// IncrReadCntIfExist note “IfExist”的含义是：如果 redis 中没有数据结构，则该方法不会执行任何操作（需要通过日志来检测并另去别的流程执行 set 方法）
	IncrReadCntIfExist(ctx context.Context, biz string, bizId int64) error
	IncrLikeCntIfExist(ctx context.Context, biz string, bizId int64) error
	DecrLikeCntIfExist(ctx context.Context, biz string, bizId int64) error
	IncrCollectCntIfExist(ctx context.Context, biz string, bizId int64) error
	DecrCollectCntIfExist(ctx context.Context, biz string, bizId int64) error
	Get(ctx context.Context, biz string, bizId int64) (domain.Interactive, error)
	Set(ctx context.Context, biz string, bizId int64, inter domain.Interactive) error

	// LikeTop note 利用 zset 实现找出点赞数top100的数据
	LikeTop(ctx context.Context, biz string) ([]domain.Interactive, error)
	IncrRankingIfExist(ctx context.Context, biz string, bizId int64) error
	SetRankingScore(ctx context.Context, biz string, bizId int64, score int64) error
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
	// 业务上：返回的 1 或 0 可以不考虑
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

// BatchSetRankingScore 将所有 interactive 存进 zset 中
func (c *RedisInteractiveCache) BatchSetRankingScore(ctx context.Context, biz string, interactives []domain.Interactive) error {
	zs := make([]redis.Z, 0, len(interactives))
	for _, interactive := range interactives {
		zs = append(zs, redis.Z{
			Score:  float64(interactive.LikeCnt),
			Member: interactive.BizId,
		})
	}
	return c.client.ZAdd(ctx, c.rankingKey(biz), zs...).Err()
}

func (c *RedisInteractiveCache) LikeTop(ctx context.Context, biz string) ([]domain.Interactive, error) {
	var start int64 = 0
	var end int64 = 99
	res, err := c.client.ZRangeWithScores(ctx, c.rankingKey(biz), start, end).Result()
	if err != nil {
		return nil, err
	}
	interactives := make([]domain.Interactive, 0, len(res))
	for _, z := range res {
		val, _ := strconv.ParseInt(z.Member.(string), 10, 64)
		interactives = append(interactives, domain.Interactive{
			BizId:   val,
			Biz:     biz,
			LikeCnt: int64(z.Score),
		})
	}
	return interactives, nil
}

func (c *RedisInteractiveCache) IncrRankingIfExist(ctx context.Context, biz string, bizId int64) error {
	_, err := c.client.Eval(ctx, luaRankingCnt, []string{c.rankingKey(biz)}, bizId).Result()
	return err
}

func (c *RedisInteractiveCache) SetRankingScore(ctx context.Context, biz string, bizId int64, score int64) error {
	_, err := c.client.Eval(ctx, luaRankingSet, []string{c.rankingKey(biz)}, bizId, score).Result()
	return err
}

func (c *RedisInteractiveCache) key(biz string, bizId int64) string {
	return fmt.Sprintf("interactive:%s:%d", biz, bizId)
}

func (c *RedisInteractiveCache) rankingKey(biz string) string {
	return fmt.Sprintf("top_100_%s", biz)
}
