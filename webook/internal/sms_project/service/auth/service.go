package auth

import (
	"context"
	"github.com/golang-jwt/jwt/v5"
	"refactor-webook/webook/internal/sms_project/service"
)

// SmsService 装饰器
type SmsService struct {
	svc service.Service
	key string
}

func NewSmsService(svc service.Service) *SmsService {
	return &SmsService{
		svc: svc,
		key: "oIft1b5qZjyLcc0zZo2UrUx5rk3KE0LvZKv73fw502oXd6vfYu1OAQvbSel8whv1",
	}
}

func (s *SmsService) Send(ctx context.Context, jwtTpl string, args []string, numbers ...string) error {
	var tc TplClaims
	_, err := jwt.ParseWithClaims(jwtTpl, &tc, func(token *jwt.Token) (interface{}, error) {
		return s.key, nil
	})
	if err != nil {
		return err
	}

	return s.svc.Send(ctx, tc.Tpl, args, numbers...)
}

type TplClaims struct {
	jwt.RegisteredClaims
	Tpl string
	// 可再添加额外字段
}
