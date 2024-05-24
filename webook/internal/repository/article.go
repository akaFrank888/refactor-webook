package repository

import (
	"context"
	"gorm.io/gorm"
	"refactor-webook/webook/internal/domain"
	"refactor-webook/webook/internal/repository/cache"
	"refactor-webook/webook/internal/repository/dao"
	"time"
)

type ArticleRepository interface {
	Create(ctx context.Context, article domain.Article) (int64, error)
	Update(ctx context.Context, article domain.Article) error
	Sync(ctx context.Context, article domain.Article) (int64, error)
}

type CachedArticleRepository struct {
	dao   dao.ArticleDao
	cache cache.ArticleCache

	// V2 写法专用  ————  在repo层同步制作库和线上库的数据（完成分发） 【不在repo开启事务】
	authorDao dao.ArticleAuthorDao
	readerDao dao.ArticleReaderDao

	// 【在repo开启事务】
	db *gorm.DB
}

func NewArticleRepository(dao dao.ArticleDao, cache cache.ArticleCache) ArticleRepository {
	return &CachedArticleRepository{
		dao:   dao,
		cache: cache,
	}
}

func NewArticleRepositoryV2(authorDao dao.ArticleAuthorDao, readerDao dao.ArticleReaderDao) *CachedArticleRepository {
	return &CachedArticleRepository{
		authorDao: authorDao,
		readerDao: readerDao,
	}
}

func (repo *CachedArticleRepository) Create(ctx context.Context, article domain.Article) (int64, error) {
	return repo.dao.Insert(ctx, repo.toPersistent(article))
}

func (repo *CachedArticleRepository) Update(ctx context.Context, article domain.Article) error {
	return repo.dao.UpdateById(ctx, repo.toPersistent(article))
}

// Sync note 【在dao层完成“发表”中制作库和线上库的分发或者叫同步，同库不同表，引入事务】
func (repo *CachedArticleRepository) Sync(ctx context.Context, article domain.Article) (int64, error) {
	return repo.dao.Sync(ctx, repo.toPersistent(article))
}

// SyncV1 note 【在repo层完成“发表”中制作库和线上库的分发或者叫同步】【V1：假定制作库和线上库是不同一个数据库，所以是不存在事务】
func (repo *CachedArticleRepository) SyncV1(ctx context.Context, article domain.Article) (int64, error) {
	article2Persistent := repo.toPersistent(article)
	var (
		err error
		id  = article.Id
	)
	if article.Id > 0 {
		err = repo.authorDao.Update(ctx, article2Persistent)
	} else {
		id, err = repo.authorDao.Create(ctx, article2Persistent)
	}
	if err != nil {
		return 0, err
	}
	// note 这句其实可以省略，因为dao层的gorm会创建个id并赋值给article，但防止dao换了别的实现，还是建议不省略
	article.Id = id
	return id, repo.readerDao.Upsert(ctx, article2Persistent)
}

// SyncV2 note 【在repo层完成“发表”中制作库和线上库的分发或者叫同步】【V2：假定制作库和线上库是同库不同表，且在repo层面完成事务，所以要在repo中跨层调用db】
func (repo *CachedArticleRepository) SyncV2(ctx context.Context, article domain.Article) (int64, error) {
	tx := repo.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return 0, tx.Error
	}
	// 防止后面的业务panic
	defer tx.Rollback()
	// note 用该 tx 封装一个基于该事务的 dao
	authorDao := dao.NewArticleGormAuthorDao(tx)
	readerDao := dao.NewArticleReaderDao(tx)

	article2Persistent := repo.toPersistent(article)
	var (
		err error
		id  = article.Id
	)
	if article.Id > 0 {
		err = authorDao.Update(ctx, article2Persistent)
	} else {
		id, err = authorDao.Create(ctx, article2Persistent)
	}
	if err != nil {
		return 0, err
	}
	article.Id = id
	err = readerDao.UpsertV2(ctx, article2Persistent)
	if err != nil {
		return 0, err
	}
	// note 手动 commit 要注意：1）“return 0, err”前都会回滚 2）虽然是 commit 后再 rollback 会报错，但不理它即可
	tx.Commit()
	return id, nil

}

func (repo *CachedArticleRepository) toPersistent(article domain.Article) dao.Article {
	return dao.Article{
		// note 因为 dao 层会处理 Ctime 和 Utime ，所以不用传
		Id:       article.Id,
		Title:    article.Title,
		Content:  article.Content,
		AuthorId: article.Author.Id,
	}
}

func (repo *CachedArticleRepository) toDomain(article dao.Article) domain.Article {
	return domain.Article{
		Id:      article.Id,
		Title:   article.Title,
		Content: article.Content,
		Author: domain.Author{
			Id: article.AuthorId,
		},
		Ctime: time.UnixMilli(article.Ctime),
		Utime: time.UnixMilli(article.Utime),
	}
}
