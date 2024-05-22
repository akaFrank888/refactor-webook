package repository

import (
	"context"
	"refactor-webook/webook/internal/domain"
	"refactor-webook/webook/internal/repository/cache"
	"refactor-webook/webook/internal/repository/dao"
	"time"
)

type ArticleRepository interface {
	Create(ctx context.Context, article domain.Article) (int64, error)
	Update(ctx context.Context, article domain.Article) error
}

type CachedArticleRepository struct {
	dao   dao.ArticleDao
	cache cache.ArticleCache
}

func NewArticleRepository(dao dao.ArticleDao, cache cache.ArticleCache) ArticleRepository {
	return &CachedArticleRepository{
		dao:   dao,
		cache: cache,
	}
}

func (repo *CachedArticleRepository) Create(ctx context.Context, article domain.Article) (int64, error) {
	return repo.dao.Insert(ctx, repo.toPersistent(article))
}

func (repo *CachedArticleRepository) Update(ctx context.Context, article domain.Article) error {
	return repo.dao.UpdateById(ctx, repo.toPersistent(article))
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
