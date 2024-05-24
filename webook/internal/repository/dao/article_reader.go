package dao

import (
	"context"
	"gorm.io/gorm"
)

type ArticleReaderDao interface {
	Upsert(ctx context.Context, article Article) error
}

type ArticleGormReaderDao struct {
	db *gorm.DB
}

func (a *ArticleGormReaderDao) Upsert(ctx context.Context, article Article) error {
	//TODO implement me
	panic("implement me")
}
func (a *ArticleGormReaderDao) UpsertV2(ctx context.Context, article Article) error {
	//TODO implement me
	panic("implement me")
}

func NewArticleReaderDao(db *gorm.DB) *ArticleGormReaderDao {
	return &ArticleGormReaderDao{
		db: db,
	}
}
