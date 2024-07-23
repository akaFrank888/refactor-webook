package repository

import (
	"context"
	"refactor-webook/webook/internal/domain"
	"refactor-webook/webook/internal/repository/cache"
)

type RankingRepository interface {
	// ReplaceTopN 更新新的 topN
	ReplaceTopN(ctx context.Context, articles []domain.Article) error
	GetTopN(ctx context.Context) ([]domain.Article, error)
}

type CachedRankingRepository struct {
	// 方案一：仅用 redis 缓存 topN
	cache cache.RankingCache

	// 方案二：用 本地缓存 + redis 缓存 topN
	// note 传入的是 结构体指针
	localCache *cache.RankingLocalCache
	redisCache *cache.RankingRedisCache
}

func NewCachedRankingRepository(cache cache.RankingCache) RankingRepository {
	return &CachedRankingRepository{cache: cache}
}

func NewCachedRankingRepositoryV1(localCache *cache.RankingLocalCache, redisCache *cache.RankingRedisCache) *CachedRankingRepository {
	// todo 我们显式要求传入 本地缓存实现和 Redis 缓存实现 （所以在 wire 时要特别处理）
	return &CachedRankingRepository{localCache: localCache, redisCache: redisCache}
}

func (c *CachedRankingRepository) ReplaceTopN(ctx context.Context, articles []domain.Article) error {
	return c.cache.Set(ctx, articles)
}

func (c *CachedRankingRepository) GetTopN(ctx context.Context) ([]domain.Article, error) {
	return c.cache.Get(ctx)
}

// ReplaceTopNV1 本地 + Redis 缓存的
// note 更新：先更新本地缓存，再更新 redis
func (c *CachedRankingRepository) ReplaceTopNV1(ctx context.Context, articles []domain.Article) error {
	// 更新本地缓存
	_ = c.localCache.Set(ctx, articles)
	return c.redisCache.Set(ctx, articles)
}

// GetTopNV1 本地 + Redis 缓存的
// note 查询：先查本地，再查 redis，最后回写本地
func (c *CachedRankingRepository) GetTopNV1(ctx context.Context) ([]domain.Article, error) {
	articles, err := c.localCache.Get(ctx)
	if err == nil {
		return articles, nil
	}
	res, err := c.redisCache.Get(ctx)
	if err != nil {
		return c.localCache.ForceGet(ctx)
	}
	return res, c.localCache.Set(ctx, res)
}
