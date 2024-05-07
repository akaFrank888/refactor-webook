package service

import (
	"context"
	"fmt"
	"math/rand"
	"refactor-webook/webook/internal/repository"
	"refactor-webook/webook/internal/service/sms"
)

type CodeService interface {
	Send(ctx context.Context, biz, phone string) error
	Verify(ctx context.Context, biz, phone, inputCode string) (bool, error)
}

var ErrCodeSendTooMany = repository.ErrCodeSendTooMany

type codeService struct {
	repo repository.CodeRepository
	sms  sms.Service
}

func NewCodeService(repo repository.CodeRepository, sms sms.Service) CodeService {
	return &codeService{
		repo: repo,
		sms:  sms,
	}
}

func (svc *codeService) Send(ctx context.Context, biz, phone string) error {
	code := svc.generateCode()
	err := svc.repo.Set(ctx, biz, phone, code)
	if err != nil {
		return err
	}
	// 调用sms服务
	const codeTplId = "1877556"
	return svc.sms.Send(ctx, codeTplId, []string{code}, phone)
}

func (svc *codeService) Verify(ctx context.Context, biz, phone, inputCode string) (bool, error) {
	ok, err := svc.repo.Verify(ctx, biz, phone, inputCode)
	if err == repository.ErrCodeVerifyTooMany {
		// note 之所以在service层指定并处理特定的错误类型，是因为业务规则 “ 一个验证码，如果已经三次验证失败，那么这个验证码就不再可用。在这种情况下，只会告诉用户输入的验证码不对，但是不会提示验证码过于频繁失败问题。”（即，对外屏蔽了验证码错误次数过多的错误，只是告诉调用者验证码不对）
		// note 而不在service层处理“发送频繁”的错误，是因为可以在handler层处理，然后返回给前端
		return false, nil
	}
	return ok, err
}

func (svc *codeService) generateCode() string {
	// 0-999999
	code := rand.Intn(1000000)
	// 06d  ==》  有前导0
	return fmt.Sprintf("%06d", code)
}
