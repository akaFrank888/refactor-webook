package repository

import (
	"context"
	"refactor-webook/webook/internal/domain"
	"refactor-webook/webook/internal/repository/cache"
)

type RankingRepository interface {
	// ReplaceTopN 更新新的 topN
	ReplaceTopN(ctx context.Context, articles []domain.Article) error
}

type CachedRankingRepository struct {
	cache cache.RankingCache
}

func NewCachedRankingRepository(cache cache.RankingCache) RankingRepository {
	return &CachedRankingRepository{cache: cache}
}

func (c *CachedRankingRepository) ReplaceTopN(ctx context.Context, articles []domain.Article) error {
	return c.cache.Set(ctx, articles)
}
