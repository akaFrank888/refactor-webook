package cache

import (
	"context"
	"errors"
	"refactor-webook/webook/internal/domain"
	"sync/atomic"
	"time"
)

// RankingLocalCache 榜单这种本地缓存的实现，可以直接用原子操作，本质上是因为我们不需要一个key-value的结构（所以就不需要类似lru.Cache那样线程安全的kv本地缓存库）
type RankingLocalCache struct {
	topN atomic.Value
	ddl  atomic.Value

	expiration time.Duration
}

func (r *RankingLocalCache) Set(ctx context.Context, articles []domain.Article) error {
	r.topN.Store(articles)
	r.ddl.Store(time.Now().Add(r.expiration))
	return nil
}

func (r *RankingLocalCache) Get(ctx context.Context) ([]domain.Article, error) {
	ddl := r.ddl.Load().(time.Time)
	arts := r.topN.Load().([]domain.Article)
	if arts == nil || ddl.Before(time.Now()) {
		return nil, errors.New("本地缓存失效（不存在或过期）")
	}
	return arts, nil

}

// ForceGet 忽略本地缓存的过期时间ddl，从而强制返回数据
func (r *RankingLocalCache) ForceGet(ctx context.Context) ([]domain.Article, error) {
	arts := r.topN.Load().([]domain.Article)
	if arts == nil {
		return nil, errors.New("本地缓存失效（不存在）")
	}
	return arts, nil
}
