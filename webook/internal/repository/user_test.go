package repository

import (
	"context"
	"database/sql"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"refactor-webook/webook/internal/domain"
	"refactor-webook/webook/internal/repository/cache"
	cachemocks "refactor-webook/webook/internal/repository/cache/mocks"
	"refactor-webook/webook/internal/repository/dao"
	daomocks "refactor-webook/webook/internal/repository/dao/mocks"
	"testing"
	"time"
)

func TestMilli(t *testing.T) {
	// note 区分 time.now() 和 time.Now().UnixMilli()
	// note 后者是取了时间戳的毫秒数，微秒纳秒等就被省略了
	// 毫秒数 int64
	nowMs := time.Now().UnixMilli()
	// 毫秒数 time
	now := time.UnixMilli(nowMs)
	t.Log(nowMs, now)
}

func TestCacheDUserRepository_FindById(t *testing.T) {
	testCases := []struct {
		name string
		mock func(ctrl *gomock.Controller) (dao.UserDao, cache.UserCache)
		// 预期输入
		ctx context.Context
		id  int64
		// 预期输出
		wantUser domain.User
		wantErr  error
	}{
		{
			name: "cache未命中，查dao回写缓存",
			mock: func(ctrl *gomock.Controller) (dao.UserDao, cache.UserCache) {
				ud := daomocks.NewMockUserDao(ctrl)
				uc := cachemocks.NewMockUserCache(ctrl)
				uc.EXPECT().Get(gomock.Any(), int64(12)).
					// 缓存未命中
					Return(domain.User{}, cache.ErrKeyNotExist)
				// 查dao
				ud.EXPECT().FindById(gomock.Any(), int64(12)).
					Return(dao.User{
						// note 把dao.User的字段全写上为了测试 toDomain 是否漏字段了
						Id: int64(12),
						Email: sql.NullString{
							String: "123@qq.com",
							Valid:  true,
						},
						Password: "123456",
						Birthday: 100,
						Resume:   "自我介绍",
						Phone: sql.NullString{
							String: "15212345678",
							Valid:  true,
						},
						Ctime: 101,
						Utime: 102,
					}, nil)
				uc.EXPECT().Set(gomock.Any(), domain.User{
					Id:       int64(12),
					Email:    "123@qq.com",
					Password: "123456",
					Birthday: time.UnixMilli(100),
					Resume:   "自我介绍",
					Phone:    "15212345678",
					Ctime:    time.UnixMilli(101),
				}).Return(nil)
				return ud, uc
			},
			ctx: context.Background(),
			id:  12,
			wantUser: domain.User{
				Id:       int64(12),
				Email:    "123@qq.com",
				Password: "123456",
				Birthday: time.UnixMilli(100),
				Resume:   "自我介绍",
				Phone:    "15212345678",
				Ctime:    time.UnixMilli(101),
			},
			wantErr: nil,
		},
		{
			name: "cache命中",
			mock: func(ctrl *gomock.Controller) (dao.UserDao, cache.UserCache) {
				ud := daomocks.NewMockUserDao(ctrl)
				uc := cachemocks.NewMockUserCache(ctrl)
				uc.EXPECT().Get(gomock.Any(), int64(12)).
					// 命中
					Return(domain.User{
						Id:       int64(12),
						Email:    "123@qq.com",
						Password: "123456",
						Birthday: time.UnixMilli(100),
						Resume:   "自我介绍",
						Phone:    "15212345678",
						Ctime:    time.UnixMilli(101),
					}, nil)
				return ud, uc
			},
			ctx: context.Background(),
			id:  12,
			wantUser: domain.User{
				Id:       int64(12),
				Email:    "123@qq.com",
				Password: "123456",
				Birthday: time.UnixMilli(100),
				Resume:   "自我介绍",
				Phone:    "15212345678",
				Ctime:    time.UnixMilli(101),
			},
			wantErr: nil,
		},
		{
			name: "缓存未命中，且dao中没找到用户",
			mock: func(ctrl *gomock.Controller) (dao.UserDao, cache.UserCache) {
				ud := daomocks.NewMockUserDao(ctrl)
				uc := cachemocks.NewMockUserCache(ctrl)
				uc.EXPECT().Get(gomock.Any(), int64(12)).
					// 缓存未命中
					Return(domain.User{}, cache.ErrKeyNotExist)
				// 查dao
				ud.EXPECT().FindById(gomock.Any(), int64(12)).
					Return(dao.User{}, dao.ErrRecordNotFound)
				return ud, uc
			},
			ctx:      context.Background(),
			id:       12,
			wantUser: domain.User{},
			wantErr:  ErrUserNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 构造repo
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ud, uc := tc.mock(ctrl)
			repo := NewUserRepository(ud, uc)

			// 调用repo
			user, err := repo.FindById(tc.ctx, tc.id)
			assert.Equal(t, tc.wantUser, user)
			assert.Equal(t, tc.wantErr, err)
		})
	}
}
