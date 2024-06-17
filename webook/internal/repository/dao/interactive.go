package dao

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

type InteractiveDao interface {
	IncrReadCnt(ctx context.Context, biz string, bizId int64) error
	InsertLikeInfo(ctx context.Context, biz string, bizId int64, uid int64) error
	InsertCancelLikeInfo(ctx context.Context, biz string, bizId int64, uid int64) error
	InsertCollectionBiz(ctx context.Context, biz string, bizId int64, uid int64, cid int64) error
	DeleteCollectionBiz(ctx context.Context, biz string, bizId int64, uid int64, cid int64) error
	GetLikeInfo(ctx context.Context, biz string, bizId int64, uid int64) (UserLikeBiz, error)
	GetCollectInfo(ctx context.Context, biz string, bizId int64, uid int64) (UserCollectionBiz, error)
	Get(ctx context.Context, biz string, bizId int64) (Interactive, error)
}

type GormInteractiveDao struct {
	db *gorm.DB
}

func NewGormInteractiveDao(db *gorm.DB) InteractiveDao {
	return &GormInteractiveDao{db: db}
}

// IncrReadCnt 增加阅读数的dao层面实现是 Upsert
func (d *GormInteractiveDao) IncrReadCnt(ctx context.Context, biz string, bizId int64) error {
	now := time.Now().UnixMilli()
	return d.db.WithContext(ctx).Clauses(clause.OnConflict{
		DoUpdates: clause.Assignments(map[string]interface{}{
			// note GORM支持SQL表达式
			"read_cnt": gorm.Expr("read_cnt + 1"),
			"utime":    now,
		}),
	}).Create(&Interactive{
		ReadCnt: 1,
		BizId:   bizId,
		Biz:     biz,
		Ctime:   now,
		Utime:   now,
	}).Error
}

// InsertLikeInfo 1. 表interactive中增加阅读数 2. 表UserLikeBiz中Upsert操作status状态为1  【两者处于同一事务中】
func (d *GormInteractiveDao) InsertLikeInfo(ctx context.Context, biz string, bizId int64, uid int64) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		// 1. 表interactive中增加阅读数
		now := time.Now().UnixMilli()
		err := tx.WithContext(ctx).Clauses(clause.OnConflict{
			DoUpdates: clause.Assignments(map[string]interface{}{
				"like_cnt": gorm.Expr("like_cnt + 1"),
				"utime":    now,
			}),
		}).Create(&Interactive{
			Biz:     biz,
			BizId:   bizId,
			LikeCnt: 1,
			Ctime:   now,
			Utime:   now,
		}).Error
		if err != nil {
			return err
		}

		// 2. 表UserLikeBiz中Upsert操作status状态为1
		return tx.WithContext(ctx).Clauses(clause.OnConflict{
			DoUpdates: clause.Assignments(map[string]interface{}{
				"Status": gorm.Expr("1"),
				"utime":  now,
			}),
		}).Create(&UserLikeBiz{
			BizId:  bizId,
			Biz:    biz,
			Uid:    uid,
			Status: 1,
			Ctime:  now,
			Utime:  now,
		}).Error
	})
}

// InsertCancelLikeInfo 1. 表UserLikeBiz中在status状态为1的前提下更改为0 2. 表interactive中减少阅读数
func (d *GormInteractiveDao) InsertCancelLikeInfo(ctx context.Context, biz string, bizId int64, uid int64) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		// 1. 表UserLikeBiz中在status状态为1的前提下更改为0
		now := time.Now().UnixMilli()

		// note 用 "AND" ，别用成了 ","
		res := tx.WithContext(ctx).Model(&UserLikeBiz{}).Where("biz = ? AND biz_id = ? AND uid = ?", biz, bizId, uid).Update("status", 0)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			// note 不管是哪种情况，都不是正常用户造成的，所以不需要分情况考虑
			return errors.New("非法操作")
		}

		// 2. 表interactive中减少阅读数
		return tx.WithContext(ctx).Model(&Interactive{}).Where("biz = ? AND biz_id = ?", biz, bizId).Updates(map[string]interface{}{
			// todo 若减成负数，则为非法请求，该怎么办？
			"like_cnt": gorm.Expr("like_cnt - 1"),
			"utime":    now,
		}).Error
	})
}

func (d *GormInteractiveDao) InsertCollectionBiz(ctx context.Context, biz string, bizId int64, uid int64, cid int64) error {
	return d.db.Transaction(func(tx *gorm.DB) error {

		now := time.Now().UnixMilli()
		err := tx.WithContext(ctx).Clauses(clause.OnConflict{
			DoUpdates: clause.Assignments(map[string]interface{}{
				"collect_cnt": gorm.Expr("collect_cnt + 1"),
				"utime":       now,
			}),
		}).Create(&Interactive{
			Biz:        biz,
			BizId:      bizId,
			CollectCnt: 1,
			Ctime:      now,
			Utime:      now,
		}).Error
		if err != nil {
			return err
		}

		return tx.WithContext(ctx).Clauses(clause.OnConflict{
			DoUpdates: clause.Assignments(map[string]interface{}{
				"Status": gorm.Expr("1"),
				"utime":  now,
				"cid":    cid,
			}),
		}).Create(&UserCollectionBiz{
			BizId:  bizId,
			Biz:    biz,
			Uid:    uid,
			Cid:    cid,
			Status: 1,
			Ctime:  now,
			Utime:  now,
		}).Error
	})
}

func (d *GormInteractiveDao) DeleteCollectionBiz(ctx context.Context, biz string, bizId int64, uid int64, cid int64) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now().UnixMilli()

		res := tx.WithContext(ctx).Model(&UserCollectionBiz{}).Where("biz = ? AND biz_id = ? AND uid = ? AND cid = ?", biz, bizId, uid, cid).Update("status", 0)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			// note 不管是哪种情况，都不是正常用户造成的，所以不需要分情况考虑
			return errors.New("非法操作")
		}

		return tx.WithContext(ctx).Model(&Interactive{}).Where("biz = ? AND biz_id = ?", biz, bizId).Updates(map[string]interface{}{
			// todo 若减成负数，则为非法请求，该怎么办？
			"collect_cnt": gorm.Expr("collect_cnt - 1"),
			"utime":       now,
		}).Error
	})
}

func (d *GormInteractiveDao) GetLikeInfo(ctx context.Context, biz string, bizId int64, uid int64) (UserLikeBiz, error) {
	var res UserLikeBiz
	return res, d.db.WithContext(ctx).Where("biz = ? AND biz_id = ? AND uid = ? AND status = ?", biz, bizId, uid, 1).First(&res).Error
}

func (d *GormInteractiveDao) GetCollectInfo(ctx context.Context, biz string, bizId int64, uid int64) (UserCollectionBiz, error) {
	var res UserCollectionBiz
	return res, d.db.WithContext(ctx).Where("biz = ? AND biz_id = ? AND uid = ?", biz, bizId, uid).First(&res).Error
}

func (d *GormInteractiveDao) Get(ctx context.Context, biz string, bizId int64) (Interactive, error) {
	var i Interactive
	return i, d.db.WithContext(ctx).Where("biz = ? AND biz_id = ?", biz, bizId).First(&i).Error
}

type Interactive struct {
	id int64 `gorm:"primaryKey, autoIncrement"`

	// note bizId更有序分度，所以bizId写在前面，建立 <biz_id,biz>的联合唯一索引
	BizId int64  `gorm:"uniqueIndex:idx_biz_id_biz"`
	Biz   string `type:"varchar(128), gorm:uniqueIndex:idx_biz_id_biz"`

	ReadCnt    int64
	LikeCnt    int64
	CollectCnt int64

	Ctime int64
	Utime int64
}

// UserLikeBiz note 在点赞功能中引入 UserLikeBiz 及其 Status 目的是：软删除（原因：1.性能方面上，更新字段比增和删记录快 2. 公司方面：公司希望保留数据）
type UserLikeBiz struct {
	id int64 `gorm:"primaryKey, autoIncrement"`

	// note <uid, biz_id, biz>联合唯一索引，加快对于该资源是否被该用户喜欢的状态的查询
	Uid   int64  `gorm:"uniqueIndex:idx_uid_biz_id_biz"`
	BizId int64  `gorm:"uniqueIndex:idx_uid_biz_id_biz"`
	Biz   string `type:"varchar(128), gorm:uniqueIndex:idx_uid_biz_id_biz"`

	Status int

	Ctime int64
	Utime int64
}

type UserCollectionBiz struct {
	id int64 `gorm:"primaryKey, autoIncrement"`

	// note 若建立的是<uid, biz_id, biz, cid>联合唯一索引，则意味着一个资源可以被收入多个收藏夹。但我们要实现的是一个资源只能被收藏一次，被收入一个收藏夹。
	Uid   int64  `gorm:"uniqueIndex:idx_uid_biz_id_biz"`
	BizId int64  `gorm:"uniqueIndex:idx_uid_biz_id_biz"`
	Biz   string `type:"varchar(128), gorm:uniqueIndex:idx_uid_biz_id_biz"`
	// note 此处并没有将 cid 建立联合索引
	Cid    int64 `gorm:"index"`
	Status int

	Ctime int64
	Utime int64
}
