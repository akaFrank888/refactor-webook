package dao

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

type ArticleDao interface {
	Insert(ctx context.Context, article Article) (int64, error)
	UpdateById(ctx context.Context, article Article) error
	Sync(ctx context.Context, article Article) (int64, error)
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

// Sync 为了简化我们的事务操作，提供了一个执行事务的闭包，只需要在闭包里面执行业务逻辑，GORM本身帮你管理了事务的生命周期。
func (dao *GormArticleDao) Sync(ctx context.Context, article Article) (int64, error) {
	var id = article.Id
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
		article.Id = id
		pubArticle := PublishedArticle(article)
		now := time.Now().UnixMilli()
		// 考虑到article可能是新建的，所以要赋值utime和ctime（即使冲突了，也不会更新ctime）
		pubArticle.Ctime = now
		pubArticle.Utime = now
		err = tx.Clauses(clause.OnConflict{
			// 为了兼容别的非mysql数据库（对mysql不起效）
			Columns: []clause.Column{{Name: "id"}},
			// 若使用mysql，则OnConflict中只有DoUpdates字段会有作用
			DoUpdates: clause.Assignments(map[string]interface{}{
				"title":   pubArticle.Title,
				"content": pubArticle.Content,
				"utime":   now,
			}),
		}).Create(&pubArticle).Error
		return err
	})
	return id, err
}

func (dao *GormArticleDao) SyncV1(ctx context.Context, article Article) (int64, error) {
	tx := dao.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return 0, tx.Error
	}
	// 防止后面的业务panic
	defer tx.Rollback()
	// note 用该 tx 再创建一个自己
	d := NewArticleDao(tx)

	var (
		err error
		id  = article.Id
	)
	// note 为什么线上库可以写成Upsert的方式，但制作库不可以？
	// ===》 因为线上库要保证只有作者才能更新自己的数据，非作者不允许（即，要进行author_id的校验这一安全验证的操作）
	//      然而sql就得写成类似这样 : insert into *** values(***) on duplicate key `content`=*** where author_id=?
	//      但mysql不支持这样的写法！！！
	if article.Id > 0 {
		err = d.UpdateById(ctx, article)
	} else {
		id, err = d.Insert(ctx, article)
	}
	if err != nil {
		return 0, err
	}
	article.Id = id
	// note 处理线上库的 Upsert
	// note 利用 article 创建一个衍生类型的对象
	pubArticle := PublishedArticle(article)
	now := time.Now().UnixMilli()
	// 考虑到article可能是新建的，所以要赋值utime和ctime（即使冲突了，也不会更新ctime）
	pubArticle.Ctime = now
	pubArticle.Utime = now
	// note Clauses是GORM提供的实现复杂sql的工具 ———— 本是一个Create语句，然后再加上了Conflict的情况下Update哪些字段
	err = tx.Clauses(clause.OnConflict{
		// 为了兼容别的非mysql数据库（对mysql不起效）
		Columns: []clause.Column{{Name: "id"}},
		// 若使用mysql，则OnConflict中只有DoUpdates字段会有作用
		// note sql : Insert into published_article (title,content,utime) values (?,?,?) on duplicate key update title="",content="",utime=""
		DoUpdates: clause.Assignments(map[string]interface{}{
			"title":   pubArticle.Title,
			"content": pubArticle.Content,
			"utime":   now,
		}),
	}).Create(&pubArticle).Error
	if err != nil {
		return 0, err
	}
	tx.Commit()
	return id, nil
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

// PublishedArticle 衍生类型，为了在“repo层将制作库和线上库进行分发，且两库满足同库不同表，采用事务处理”时，PublishedArticle代表读者读取的表
type PublishedArticle Article
type PublishedArticleV1 struct {
	Article
}
