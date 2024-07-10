package repository

import (
	"context"
	"refactor-webook/webook/internal/sms_project/domain"
	"refactor-webook/webook/internal/sms_project/repository/dao"
	"refactor-webook/webook/pkg/sqlx"
)

var ErrWaitingSmsNotFound = dao.ErrWaitingSmsNotFound

type AsyncSmsRepository interface {
	Add(ctx context.Context, sms domain.AsyncSms) error
	PreemptWaitingSms(ctx context.Context) (domain.AsyncSms, error)
	ReportScheduleResult(ctx context.Context, id int64, success bool) error
}

type asyncSmsRepository struct {
	dao dao.AsyncSmsDao
}

func (a *asyncSmsRepository) Add(ctx context.Context, sms domain.AsyncSms) error {
	err := a.dao.Insert(ctx, dao.AsyncSms{
		Id: sms.Id,
		Config: sqlx.JsonColumn[dao.SmsConfig]{
			Val: dao.SmsConfig{
				TplId:   sms.TplId,
				Args:    sms.Args,
				Numbers: sms.Numbers,
			},
			Valid: true,
		},
		RetryMax: sms.RetryMax,
	})
	return err
}

func (a *asyncSmsRepository) PreemptWaitingSms(ctx context.Context) (domain.AsyncSms, error) {
	waitingSms, err := a.dao.GetWaitingSms(ctx)
	if err != nil {
		return domain.AsyncSms{}, err
	}
	return domain.AsyncSms{
		Id:       waitingSms.Id,
		TplId:    waitingSms.Config.Val.TplId,
		Args:     waitingSms.Config.Val.Args,
		Numbers:  waitingSms.Config.Val.Numbers,
		RetryMax: waitingSms.RetryMax,
	}, nil
}

// ReportScheduleResult 抢占行级锁成功的情况下，Send() 成功与否
func (a *asyncSmsRepository) ReportScheduleResult(ctx context.Context, id int64, success bool) error {
	if success {
		return a.dao.MarkSuccess(ctx, id)
	}
	return a.dao.MarkFailed(ctx, id)
}
