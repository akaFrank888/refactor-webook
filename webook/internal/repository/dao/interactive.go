package dao

import (
	"context"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

type InteractiveDao interface {
	IncrReadCnt(ctx context.Context, biz string, bizId int64) error
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
		bizId: bizId,
		biz:   biz,
		Ctime: now,
		Utime: now,
	}).Error
}

type Interactive struct {
	id int64 `gorm:"primaryKey, autoIncrement"`

	// note bizId更有序分度，所以bizId写在前面，建立 <biz_id,biz>的联合唯一索引
	bizId int64  `gorm:"uniqueIndex:idx_biz_id_biz"`
	biz   string `type:"varchar(128), gorm:uniqueIndex:idx_biz_id_biz"`

	readCnt    int64
	LikeCnt    int64
	CollectCnt int64

	Ctime int64
	Utime int64
}
