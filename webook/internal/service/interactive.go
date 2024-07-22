package service

import (
	"context"
	"golang.org/x/sync/errgroup"
	"refactor-webook/webook/internal/domain"
	"refactor-webook/webook/internal/repository"
)

//go:generate mockgen -source=./interactive.go -package=svcmocks -destination=./mocks/interactive.mock.go InteractiveService
type InteractiveService interface {
	IncrReadCnt(ctx context.Context, biz string, bizId int64) error
	Like(ctx context.Context, biz string, bizId int64, uid int64) error
	CancelLike(ctx context.Context, biz string, id int64, uid int64) error
	Collect(ctx context.Context, biz string, bizId int64, uid int64, cid int64) error
	CancelCollect(ctx context.Context, biz string, bizId int64, uid int64, cid int64) error
	Get(ctx context.Context, biz string, bizId int64, uid int64) (domain.Interactive, error)
	// GetByIds 用于榜单模型
	GetByIds(ctx context.Context, biz string, bizIds []int64) (map[int64]domain.Interactive, error)
}

type interactiveService struct {
	repo repository.InteractiveRepository
}

func NewInteractiveService(repo repository.InteractiveRepository) InteractiveService {
	return &interactiveService{repo: repo}
}

func (i *interactiveService) IncrReadCnt(ctx context.Context, biz string, bizId int64) error {
	return i.repo.IncrReadCnt(ctx, biz, bizId)
}

func (i *interactiveService) Like(ctx context.Context, biz string, bizId int64, uid int64) error {
	return i.repo.IncrLike(ctx, biz, bizId, uid)
}

func (i *interactiveService) CancelLike(ctx context.Context, biz string, id int64, uid int64) error {
	return i.repo.DecrLike(ctx, biz, id, uid)
}

func (i *interactiveService) Collect(ctx context.Context, biz string, bizId int64, uid int64, cid int64) error {
	return i.repo.AddCollectionItem(ctx, biz, bizId, uid, cid)
}

func (i *interactiveService) CancelCollect(ctx context.Context, biz string, bizId int64, uid int64, cid int64) error {
	return i.repo.RemoveCollectionItem(ctx, biz, bizId, uid, cid)
}
func (i *interactiveService) Get(ctx context.Context, biz string, bizId int64, uid int64) (domain.Interactive, error) {
	// note 为什么先取阅读量点赞量和收藏量，再并行取是否点赞和收藏？而不是直接并行取这三部分？===》业务上认为到三个量是该方法的核心，如果该部分报错就没有必要并行取是否点赞和收藏了
	inter, err := i.repo.Get3Cnt(ctx, biz, bizId)
	if err != nil {
		return domain.Interactive{}, err
	}
	var eg errgroup.Group
	eg.Go(func() error {
		var er error
		inter.Liked, er = i.repo.Liked(ctx, biz, bizId, uid)
		return er
	})
	eg.Go(func() error {
		var er error
		inter.Collected, er = i.repo.Collected(ctx, biz, bizId, uid)
		return er
	})

	// note 如果要考虑降级策略（缓解 mysql 和 redis 的压力），获取 interactive 时报错，甚至可以不返回 err 返回 nil ，因为不影响 article 的核心业务
	return inter, eg.Wait()
}

func (i *interactiveService) GetByIds(ctx context.Context, biz string, bizIds []int64) (map[int64]domain.Interactive, error) {
	inters, err := i.repo.GetByIds(ctx, biz, bizIds)
	if err != nil {
		return nil, err
	}
	res := make(map[int64]domain.Interactive, len(inters))
	for _, inter := range inters {
		res[inter.BizId] = inter
	}
	return res, nil
}
