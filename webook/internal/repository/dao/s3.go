package dao

import (
	"bytes"
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strconv"
	"time"
)

type ArticleS3DAO struct {
	GormArticleDao
	oss *minio.Client
}

func NewArticleS3DAO(db *gorm.DB, oss *minio.Client) *ArticleS3DAO {
	return &ArticleS3DAO{
		GormArticleDao: GormArticleDao{db: db},
		oss:            oss,
	}
}
func (dao *ArticleS3DAO) SyncStatus(ctx *gin.Context, uid int64, aid int64, status uint8) error {
	now := time.Now().UnixMilli()
	err := dao.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// updates 语句需要接收res来判断是否更新成功
		res := tx.Model(&Article{}).
			Where("id = ? and author_id = ?", aid, uid).
			Updates(map[string]any{
				"status": status,
				"utime":  now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return errors.New("文章id或者作者id有误，更新失败")
		}
		// 再更新线上库的status
		return tx.Model(&PublishedArticleV2{}).
			Where("id = ?", uid).
			Updates(map[string]any{
				"utime":  now,
				"status": status,
			}).Error
	})
	if err != nil {
		return err
	}
	// note 在设置为“仅自己可见”的时候，要把数据从OSS 里面删除掉
	const statusPrivate = 3
	if status == statusPrivate {
		err = dao.oss.RemoveObject(context.Background(), "my-bucket", strconv.FormatInt(aid, 10), minio.RemoveObjectOptions{})
	}
	return err
}

func (dao *ArticleS3DAO) Sync(ctx context.Context, article Article) (int64, error) {
	var id = article.Id
	// 制作库
	err := dao.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		d := NewArticleDao(tx)
		var err error
		if article.Id > 0 {
			err = d.UpdateById(ctx, article)
		} else {
			id, err = d.Insert(ctx, article)
		}
		if err != nil {
			return err
		}

		// 线上库
		now := time.Now().UnixMilli()
		article.Id = id
		pubArticle := PublishedArticleV2{
			Id:       article.Id,
			Title:    article.Title,
			AuthorId: article.AuthorId,
			Ctime:    now,
			Utime:    now,
			Status:   article.Status,
		}
		// 考虑到article可能是新建的，所以要赋值utime和ctime（即使冲突了，也不会更新ctime）
		pubArticle.Ctime = now
		pubArticle.Utime = now
		err = tx.Clauses(clause.OnConflict{
			// 为了兼容别的非mysql数据库（对mysql不起效）
			Columns: []clause.Column{{Name: "id"}},
			// 若使用mysql，则OnConflict中只有DoUpdates字段会有作用
			DoUpdates: clause.Assignments(map[string]interface{}{
				"title":  pubArticle.Title,
				"utime":  now,
				"status": pubArticle.Status,
			}),
		}).Create(&pubArticle).Error
		return err
	})

	if err != nil {
		return 0, err
	}

	// 将article的content同步到oss上
	_, err = dao.oss.PutObject(context.Background(), "my-bucket", strconv.FormatInt(article.Id, 10), bytes.NewReader([]byte(article.Content)), -1, minio.PutObjectOptions{
		ContentType: "text/plain;charset=utf-8",
	})
	return id, err
}

type PublishedArticleV2 struct {
	Id      int64  `gorm:"primary_key, autoIncrement"`
	Title   string `gorm:"type=varchar(4096)"`
	Content string `gorm:"type=BLOB"`
	// 根据创作者id来查询
	AuthorId int64 `gorm:"index"`
	Ctime    int64
	Utime    int64

	Status uint8
}
