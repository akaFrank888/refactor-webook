package repository

import (
	"context"
	"refactor-webook/webook/internal/domain"
	"refactor-webook/webook/internal/repository/cache"
	"refactor-webook/webook/internal/repository/dao"
	"refactor-webook/webook/pkg/logger"
)

type InteractiveRepository interface {
	IncrReadCnt(ctx context.Context, biz string, bizId int64) error
	BatchIncrReadCnt(ctx context.Context, biz []string, bizId []int64) error
	IncrLike(ctx context.Context, biz string, bizId int64, uid int64) error
	DecrLike(ctx context.Context, biz string, bizId int64, uid int64) error
	AddCollectionItem(ctx context.Context, biz string, bizId int64, uid int64, cid int64) error
	RemoveCollectionItem(ctx context.Context, biz string, bizId int64, uid int64, cid int64) error
	Get3Cnt(ctx context.Context, biz string, bizId int64) (domain.Interactive, error)
	Liked(ctx context.Context, biz string, bizId int64, uid int64) (bool, error)
	Collected(ctx context.Context, biz string, bizId int64, uid int64) (bool, error)
}

type CachedInteractiveRepository struct {
	dao   dao.InteractiveDao
	cache cache.InteractiveCache

	l logger.LoggerV1
}

func NewCachedInteractiveRepository(dao dao.InteractiveDao, cache cache.InteractiveCache) InteractiveRepository {
	return &CachedInteractiveRepository{dao: dao, cache: cache}
}

func (c *CachedInteractiveRepository) IncrLike(ctx context.Context, biz string, bizId int64, uid int64) error {
	// note 关于区别于阅读数+1的命名：因为dao层除了实现点赞数+1，还要标记已赞状态
	err := c.dao.InsertLikeInfo(ctx, biz, bizId, uid)
	if err != nil {
		return err
	}
	return c.cache.IncrLikeCntIfExist(ctx, biz, bizId)
}

func (c *CachedInteractiveRepository) DecrLike(ctx context.Context, biz string, bizId int64, uid int64) error {
	err := c.dao.InsertCancelLikeInfo(ctx, biz, bizId, uid)
	if err != nil {
		return err
	}
	return c.cache.DecrLikeCntIfExist(ctx, biz, bizId)
}

func (c *CachedInteractiveRepository) AddCollectionItem(ctx context.Context, biz string, bizId int64, uid int64, cid int64) error {
	err := c.dao.InsertCollectionBiz(ctx, biz, bizId, uid, cid)
	if err != nil {
		return err
	}
	return c.cache.IncrCollectCntIfExist(ctx, biz, bizId)
}

func (c *CachedInteractiveRepository) RemoveCollectionItem(ctx context.Context, biz string, bizId int64, uid int64, cid int64) error {
	err := c.dao.DeleteCollectionBiz(ctx, biz, bizId, uid, cid)
	if err != nil {
		return err
	}
	return c.cache.DecrCollectCntIfExist(ctx, biz, bizId)
}

func (c *CachedInteractiveRepository) Get3Cnt(ctx context.Context, biz string, bizId int64) (domain.Interactive, error) {
	res, err := c.cache.Get(ctx, biz, bizId)
	if err == nil {
		return res, nil
	}
	interDao, err := c.dao.Get(ctx, biz, bizId)
	inter := c.toDomain(interDao)
	if err != nil {
		return domain.Interactive{}, err
	}
	// 回写缓存
	go func() {
		er := c.cache.Set(ctx, biz, bizId, inter)
		if er != nil {
			c.l.Error("回写缓存失败",
				logger.Error(er),
				logger.String("biz", biz),
				logger.Int64("bizId", bizId))
		}
	}()
	return inter, nil

	// note 该方法中涉及的 查缓存+查db+回写缓存 会存在数据一致性问题，但我们基于对阅读点赞收藏的interactive理解，选择不解决数据一致性的问题（因为对于文章，并发高的文章不差一两个数量的误差；并发低的文章，不太会出现一致性问题）
	// note tip : 缓存一致性问题虽然重要，但并不是所有场景都需要去解决。反过来说，需要彻底解决数据一致性问题的场景，就不该用缓存
}

// Liked 没用缓存
func (c *CachedInteractiveRepository) Liked(ctx context.Context, biz string, bizId int64, uid int64) (bool, error) {
	_, err := c.dao.GetLikeInfo(ctx, biz, bizId, uid)
	// note 因为dao中的sql中有status=1的条件，所以此处只要不报错就是true，即like了
	switch err {
	case nil:
		return true, nil
	case dao.ErrRecordNotFound:
		return false, nil
	default:
		return false, err
	}
}

func (c *CachedInteractiveRepository) IncrReadCnt(ctx context.Context, biz string, bizId int64) error {
	err := c.dao.IncrReadCnt(ctx, biz, bizId)
	if err != nil {
		return err
	}
	return c.cache.IncrReadCntIfExist(ctx, biz, bizId)
}

func (c *CachedInteractiveRepository) BatchIncrReadCnt(ctx context.Context, biz []string, bizId []int64) error {
	err := c.dao.BatchIncrReadCnt(ctx, biz, bizId)
	if err != nil {
		return err
	}
	go func() {
		for i := 0; i < len(biz); i++ {
			er := c.cache.IncrReadCntIfExist(ctx, biz[i], bizId[i])
			if er != nil {
				// 记录日志
			}
		}
	}()
	return nil
}

// Collected 没用缓存
func (c *CachedInteractiveRepository) Collected(ctx context.Context, biz string, bizId int64, uid int64) (bool, error) {
	_, err := c.dao.GetCollectInfo(ctx, biz, bizId, uid)
	// note 因为dao中的sql中有status=1的条件，所以此处只要不报错就是true，即like了
	switch err {
	case nil:
		return true, nil
	case dao.ErrRecordNotFound:
		return false, nil
	default:
		return false, err
	}
}

func (c *CachedInteractiveRepository) toDomain(i dao.Interactive) domain.Interactive {
	return domain.Interactive{
		ReadCnt:    i.ReadCnt,
		LikeCnt:    i.LikeCnt,
		CollectCnt: i.CollectCnt,
	}
}
