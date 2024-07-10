package dao

import (
	"context"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"refactor-webook/webook/pkg/sqlx"
	"time"
)

var ErrWaitingSmsNotFound = gorm.ErrRecordNotFound

type AsyncSms struct {
	Id       int64
	Config   sqlx.JsonColumn[SmsConfig]
	RetryMax int64
	Status   int8
	Ctime    int64
	Utime    int64 `gorm:"index"`
}

type SmsConfig struct {
	TplId   string
	Args    []string
	Numbers []string
}

const (
	asyncStatusWait = iota
	asyncStatusFailed
	asyncStatusSuccess
)

type AsyncSmsDao interface {
	Insert(ctx context.Context, sms AsyncSms) error
	GetWaitingSms(ctx context.Context) (AsyncSms, error)
	MarkSuccess(ctx context.Context, id int64) error
	MarkFailed(ctx context.Context, id int64) error
}

type GormAsyncSmsDao struct {
	db *gorm.DB
}

func (g *GormAsyncSmsDao) Insert(ctx context.Context, sms AsyncSms) error {
	// 设置 status 和 时间
	sms.Status = asyncStatusWait
	now := time.Now().UnixMilli()
	sms.Ctime = now
	sms.Utime = now

	return g.db.WithContext(ctx).Create(&sms).Error
}

// GetWaitingSms 先 select 一个 waitingSms, 再 update 它的 retry_cnt 和 Utime（在一个事务中完成，使用行级锁）
func (g *GormAsyncSmsDao) GetWaitingSms(ctx context.Context) (AsyncSms, error) {
	var sms AsyncSms
	// note 行级锁需要在事务中使用，因为当事务提交了，锁就会被释放
	err := g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UnixMilli()
		endTime := now - time.Minute.Milliseconds()
		// note select for update 行级锁中的独占锁
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("utime < ? and status = ?", endTime, asyncStatusWait).
			First(&sms).Error
		if err != nil {
			return err
		}

		// 更新 utime 和 retry_cnt
		err = tx.Model(&AsyncSms{}).Where("id = ?", sms.Id).Updates(map[string]any{
			"utime":     now,
			"retry_cnt": gorm.Expr("retry_cnt + 1"),
		}).Error // note 不需要检查 res.RowsAffected == 0，因为记录一定存在

		return err
	})
	return sms, err
}

// MarkSuccess 抢占成功，发送成功 ==> 修改 status 和 utime
func (g *GormAsyncSmsDao) MarkSuccess(ctx context.Context, id int64) error {
	return g.db.WithContext(ctx).Where("id = ?", id).Updates(map[string]any{
		"status": asyncStatusSuccess,
		"utime":  time.Now().UnixMilli(),
	}).Error

}

// MarkFailed 抢占成功，发送失败 ==> 只有到了重试最大次数再更新 status 和 utime
func (g *GormAsyncSmsDao) MarkFailed(ctx context.Context, id int64) error {
	return g.db.WithContext(ctx).Where("id = ? and retry_cnt >= `retry_max`", id).
		// note `retry_max` 是当前记录的一个字段
		Updates(map[string]any{
			"status": asyncStatusFailed,
			"utime":  time.Now().UnixMilli(),
		}).Error
}
