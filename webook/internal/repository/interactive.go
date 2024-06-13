package repository

import (
	"context"
	"refactor-webook/webook/internal/repository/cache"
	"refactor-webook/webook/internal/repository/dao"
)

type InteractiveRepository interface {
	IncrReadCnt(ctx context.Context, biz string, bizId int64) error
}

type CachedInteractiveRepository struct {
	dao   dao.InteractiveDao
	cache cache.InteractiveCache
}

func NewCachedInteractiveRepository(dao dao.InteractiveDao, cache cache.InteractiveCache) InteractiveRepository {
	return &CachedInteractiveRepository{dao: dao, cache: cache}
}

func (c *CachedInteractiveRepository) IncrReadCnt(ctx context.Context, biz string, bizId int64) error {
	err := c.dao.IncrReadCnt(ctx, biz, bizId)
	if err != nil {
		return err
	}
	return c.cache.IncrReadCntIfExist(ctx, biz, bizId)
}
