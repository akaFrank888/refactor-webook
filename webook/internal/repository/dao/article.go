package dao

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"time"
)

type ArticleDao interface {
	Insert(ctx context.Context, article Article) (int64, error)
	UpdateById(ctx context.Context, article Article) error
}

type GormArticleDao struct {
	db *gorm.DB
}

func NewArticleDao(db *gorm.DB) ArticleDao {
	return &GormArticleDao{
		db: db,
	}
}

func (dao *GormArticleDao) Insert(ctx context.Context, article Article) (int64, error) {
	now := time.Now().UnixMilli()
	article.Utime = now
	article.Ctime = now

	err := dao.db.WithContext(ctx).Create(&article).Error
	// note 虽然插入的时候没有 article.Id ，但是执行完上面的 sql 后，article.Id 就被填进去了
	return article.Id, err
}

func (dao *GormArticleDao) UpdateById(ctx context.Context, article Article) error {
	now := time.Now().UnixMilli()
	// note 校对 author_id 的目的是 防止用户修改别人的文章
	res := dao.db.WithContext(ctx).Model(&Article{}).Where("id = ? and author_id = ?", article.Id, article.AuthorId).Updates(map[string]any{
		"title":   article.Title,
		"content": article.Content,
		"utime":   now,
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		// note 不管是哪种情况，都不是正常用户造成的，所以不需要分情况考虑
		return errors.New("文章id或者作者id有误，更新失败")
	}
	return nil
}

type Article struct {
	Id      int64  `gorm:"primary_key, autoIncrement"`
	Title   string `gorm:"type=varchar(4096)"`
	Content string `gorm:"type=BLOB"`
	// 根据创作者id来查询
	AuthorId int64 `gorm:"index"`
	Ctime    int64
	Utime    int64
}
