package service

import (
	"context"
	"errors"
	"refactor-webook/webook/internal/domain"
	"refactor-webook/webook/internal/repository"
	"refactor-webook/webook/pkg/logger"
)

type ArticleService interface {
	Save(ctx context.Context, article domain.Article) (int64, error)
	Publish(ctx context.Context, article domain.Article) (int64, error)
}

type articleService struct {
	repo repository.ArticleRepository

	// V1 写法专用
	authorRepo repository.ArticleAuthorRepository
	readerRepo repository.ArticleReaderRepository
	l          logger.LoggerV1
}

func NewArticleService(repo repository.ArticleRepository) ArticleService {
	return &articleService{
		repo: repo,
	}
}

func NewArticleServiceV1(authorRepo repository.ArticleAuthorRepository, readerRepo repository.ArticleReaderRepository,
	l logger.LoggerV1) *articleService {
	return &articleService{
		authorRepo: authorRepo,
		readerRepo: readerRepo,
		l:          l,
	}
}

// Save note  判断article是新建还是更新在哪里进行分发？ ==》 service 层的 save()
// Save note 如何判断article是新建还是更新？ ==》 传入了 id 就是更新，没传就是新建 ==》 update()只用返回 error ，create()还要返回id
func (svc *articleService) Save(ctx context.Context, article domain.Article) (int64, error) {
	if article.Id > 0 {
		// 是update
		return article.Id, svc.Update(ctx, article)
	}
	return svc.repo.Create(ctx, article)
}

func (svc *articleService) Update(ctx context.Context, article domain.Article) error {
	return svc.repo.Update(ctx, article)
}

func (svc *articleService) Publish(ctx context.Context, article domain.Article) (int64, error) {
	return svc.repo.Create(ctx, article)
}

// PublishV1 note 1. 先新建/更新到“操作库”，再保存到“线上库” 2. 约定 : 操作库喝线上库的帖子 id 是相同的
func (svc *articleService) PublishV1(ctx context.Context, article domain.Article) (int64, error) {
	var (
		id  = article.Id
		err error
	)
	// 根据article中的id判断是新建还是更新
	if article.Id > 0 {
		err = svc.authorRepo.Update(ctx, article)
	} else {
		id, err = svc.authorRepo.Create(ctx, article)
	}
	if err != nil {
		// 若制作库出错，就直接返回
		return 0, err
	}
	// 考虑到可能会进入 id > 0 的分支，所以反向赋值
	article.Id = id

	// note 对部分失败的考虑（操作库执行成功，线上库执行失败）
	// 1. 为什么不引入事务？===》 Service这一层不适合开启事务且不一定能开启事务，原因有二：
	//    a. 因为Service层看不到repo的存储方式，更无法执行事务
	//    b. 制作库和线上库也不一定能开启事务，因为开启事务的条件必须是同库不同表
	// 2. 为什么引入重试？  （一般在实践中是不着急引入重试的，我得上线观察一下这边会不会经常出现部分失败，再考虑要不要引入。）
	//    a. 对于创作者，他能够看到发表失败的响应，所以他可以考虑重试
	//    b. 对于读者，他还是能看到上次发表的内容的

	for i := 0; i < 3; i++ {
		err = svc.readerRepo.Save(ctx, article)
		if err != nil {
			svc.l.Error("部分失败：保存数据到线上库失败",
				logger.Int64("aid", article.Id),
				logger.Error(err))
		} else {
			return id, nil
		}
	}
	svc.l.Error("部分失败：保存数据到线上库三次重试后仍失败",
		logger.Int64("aid", article.Id),
		logger.Error(err))

	return id, errors.New("保存到线上库失败，次数耗尽")
}
