package cache

import (
	"context"
	"encoding/json"
	"github.com/redis/go-redis/v9"
	"refactor-webook/webook/internal/domain"
	"time"
)

type RankingCache interface {
	Set(ctx context.Context, articles []domain.Article) error
	Get(ctx context.Context) ([]domain.Article, error)
}

type RankingRedisCache struct {
	client     redis.Cmdable
	key        string
	expiration time.Duration
}

func NewRankingRedisCache(client redis.Cmdable) RankingCache {
	return &RankingRedisCache{client: client, key: "ranking:top_n", expiration: time.Minute * 3}
}

func (r *RankingRedisCache) Set(ctx context.Context, articles []domain.Article) error {
	// note 缓存榜单的 top100 文章时，不要缓存文章内容
	for _, article := range articles {
		article.Content = article.Abstract()
	}

	// 直接序列化 []article ，存入 redis
	res, err := json.Marshal(articles)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, r.key, res, r.expiration).Err()
}

func (r *RankingRedisCache) Get(ctx context.Context) ([]domain.Article, error) {
	var res []domain.Article

	val, err := r.client.Get(ctx, r.key).Bytes()
	if err != nil {
		return nil, err
	}
	return res, json.Unmarshal(val, &res)
}
