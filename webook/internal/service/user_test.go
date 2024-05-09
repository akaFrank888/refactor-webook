package service

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"
	"refactor-webook/webook/internal/domain"
	"refactor-webook/webook/internal/repository"
	repomocks "refactor-webook/webook/internal/repository/mocks"
	"testing"
)

func TestPasswordEncrypt(t *testing.T) {
	password := []byte("123456#hello")
	encrypted, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	assert.NoError(t, err)
	println(string(encrypted))
	err = bcrypt.CompareHashAndPassword(encrypted, []byte("123456#hello"))
	assert.NoError(t, err)
}

func TestUserService_Login(t *testing.T) {
	testcases := []struct {
		name string
		mock func(ctrl *gomock.Controller) repository.UserRepository
		// 输入
		ctx      context.Context
		email    string
		password string
		// 输出
		wantErr  error
		wantUser domain.User
	}{
		{
			name: "登录成功",
			mock: func(ctrl *gomock.Controller) repository.UserRepository {
				repo := repomocks.NewMockUserRepository(ctrl)
				repo.EXPECT().FindByEmail(gomock.Any(), "123@qq.com").
					Return(domain.User{
						Email:    "123@qq.com",
						Password: "$2a$10$GxePYuIWdG0EREj2TrcUkucrdpOWH1ggbrqD7TTpEi4/S2vpfx50S",
						Phone:    "123444444",
					}, nil)
				return repo
			},
			email:    "123@qq.com",
			password: "123456#hello",

			wantUser: domain.User{
				Email:    "123@qq.com",
				Password: "$2a$10$GxePYuIWdG0EREj2TrcUkucrdpOWH1ggbrqD7TTpEi4/S2vpfx50S",
				Phone:    "123444444",
			},
			wantErr: nil,
		},
		{
			name: "用户未找到",
			mock: func(ctrl *gomock.Controller) repository.UserRepository {
				repo := repomocks.NewMockUserRepository(ctrl)
				repo.EXPECT().FindByEmail(gomock.Any(), "123@qq.com").
					Return(domain.User{}, repository.ErrUserNotFound)
				return repo
			},
			email:    "123@qq.com",
			password: "123456#hello",

			wantUser: domain.User{},
			wantErr:  ErrInvalidEmailOrPassword,
		},
		{
			name: "系统错误",
			mock: func(ctrl *gomock.Controller) repository.UserRepository {
				repo := repomocks.NewMockUserRepository(ctrl)
				repo.EXPECT().FindByEmail(gomock.Any(), "123@qq.com").
					Return(domain.User{}, errors.New("db错误"))
				return repo
			},
			email:    "123@qq.com",
			password: "123456#hello",

			wantUser: domain.User{},
			wantErr:  errors.New("db错误"),
		},
		{
			name: "密码错误",
			mock: func(ctrl *gomock.Controller) repository.UserRepository {
				repo := repomocks.NewMockUserRepository(ctrl)
				repo.EXPECT().FindByEmail(gomock.Any(), "123@qq.com").
					Return(domain.User{
						Email:    "123@qq.com",
						Password: "$2a$10$GxePYuIWdG0EREj2TrcUkucrdpOWH1ggbrqD7TTpEi4/S2vpfx50S",
						Phone:    "123444444",
					}, nil)
				return repo
			},
			email: "123@qq.com",
			// 传入一个错误的密码
			password: "123456#hello111",

			wantUser: domain.User{},
			wantErr:  ErrInvalidEmailOrPassword,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			// 先构造svc
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			repo := tc.mock(ctrl)
			userSvc := NewUserService(repo)

			// 调用login
			user, err := userSvc.Login(tc.ctx, tc.email, tc.password)
			assert.Equal(t, tc.wantErr, err)
			assert.Equal(t, tc.wantUser, user)
		})
	}
}
