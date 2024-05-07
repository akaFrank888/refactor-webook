package repository

import (
	"context"
	"refactor-webook/webook/internal/repository/cache"
)

type CodeRepository interface {
	Set(ctx context.Context, biz, phone, code string) error
	Verify(ctx context.Context, biz, phone, code string) (bool, error)
}

var (
	ErrCodeSendTooMany   = cache.ErrCodeSendTooMany
	ErrCodeVerifyTooMany = cache.ErrCodeVerifyTooMany
)

func (r *CachedCodeRepository) Set(ctx context.Context, biz, phone, code string) error {
	return r.cache.Set(ctx, biz, phone, code)
}

func (r *CachedCodeRepository) Verify(ctx context.Context, biz, phone, code string) (bool, error) {
	return r.cache.Verify(ctx, biz, phone, code)
}

type CachedCodeRepository struct {
	cache cache.CodeCache
}

func NewCodeRepository(cache cache.CodeCache) CodeRepository {
	return &CachedCodeRepository{
		cache: cache,
	}
}
