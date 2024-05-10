package cache

import (
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"refactor-webook/webook/internal/repository/cache/redismocks"
	"testing"
)

func TestRedisCodeCache_Set(t *testing.T) {
	keyFunc := func(biz, phone string) string {
		return fmt.Sprintf("phone_code:%s:%s", biz, phone)
	}

	testCases := []struct {
		name string
		mock func(ctrl *gomock.Controller) redis.Cmdable
		// 预期输入
		ctx   context.Context
		biz   string
		phone string
		code  string
		// 预期输出
		wantErr error
	}{
		{
			name: "设置成功",
			mock: func(ctrl *gomock.Controller) redis.Cmdable {
				cmd := redismocks.NewMockCmdable(ctrl)
				// note 因为 Eval() 返回的是 Cmd，所以要在这构造好一个cmd，再让下面 return 这个cmd
				cmdRes := redis.NewCmd(context.Background())
				// note 因为还要调用 int()
				cmdRes.SetErr(nil)
				cmdRes.SetVal(int64(0))
				cmd.EXPECT().Eval(gomock.Any(), luaSetCode,
					// note []string{} 和 []any{} 参数类型要对应上，不然会报错
					[]string{keyFunc("test", "1231434234")},
					// note Eval的最后一个参数类型是不定参数，所以要传切片，不要直接传string！
					[]any{"123456"}).Return(cmdRes)
				return cmd
			},
			ctx:     context.Background(),
			biz:     "test",
			phone:   "1231434234",
			code:    "123456",
			wantErr: nil,
		},
		{
			name: "redis返回err",
			mock: func(ctrl *gomock.Controller) redis.Cmdable {
				cmd := redismocks.NewMockCmdable(ctrl)
				// note 因为 Eval() 返回的是 Cmd，所以要在这构造好一个cmd，再让下面 return 这个cmd
				cmdRes := redis.NewCmd(context.Background())
				// note 因为还要调用 int()
				cmdRes.SetErr(errors.New("redis err"))
				cmd.EXPECT().Eval(gomock.Any(), luaSetCode,
					// note []string{} 和 []any{} 参数类型要对应上，不然会报错
					[]string{keyFunc("test", "1231434234")},
					// note Eval的最后一个参数类型是不定参数，所以要传切片，不要直接传string！
					[]any{"123456"}).Return(cmdRes)
				return cmd
			},
			ctx:     context.Background(),
			biz:     "test",
			phone:   "1231434234",
			code:    "123456",
			wantErr: errors.New("redis err"),
		},
		{
			name: "验证码存在，但是没有过期时间",
			mock: func(ctrl *gomock.Controller) redis.Cmdable {
				cmd := redismocks.NewMockCmdable(ctrl)
				// note 因为 Eval() 返回的是 Cmd，所以要在这构造好一个cmd，再让下面 return 这个cmd
				cmdRes := redis.NewCmd(context.Background())
				// note 因为还要调用 int()
				cmdRes.SetVal(int64(-2))
				cmd.EXPECT().Eval(gomock.Any(), luaSetCode,
					// note []string{} 和 []any{} 参数类型要对应上，不然会报错
					[]string{keyFunc("test", "1231434234")},
					// note Eval的最后一个参数类型是不定参数，所以要传切片，不要直接传string！
					[]any{"123456"}).Return(cmdRes)
				return cmd
			},
			ctx:     context.Background(),
			biz:     "test",
			phone:   "1231434234",
			code:    "123456",
			wantErr: errors.New("验证码存在，但是没有过期时间"),
		},
		{
			name: "验证码发送太频繁",
			mock: func(ctrl *gomock.Controller) redis.Cmdable {
				cmd := redismocks.NewMockCmdable(ctrl)
				// note 因为 Eval() 返回的是 Cmd，所以要在这构造好一个cmd，再让下面 return 这个cmd
				cmdRes := redis.NewCmd(context.Background())
				// note 因为还要调用 int()
				cmdRes.SetVal(int64(-1))
				cmd.EXPECT().Eval(gomock.Any(), luaSetCode,
					// note []string{} 和 []any{} 参数类型要对应上，不然会报错
					[]string{keyFunc("test", "1231434234")},
					// note Eval的最后一个参数类型是不定参数，所以要传切片，不要直接传string！
					[]any{"123456"}).Return(cmdRes)
				return cmd
			},
			ctx:     context.Background(),
			biz:     "test",
			phone:   "1231434234",
			code:    "123456",
			wantErr: ErrCodeSendTooMany,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			cmd := tc.mock(ctrl)
			cc := NewCodeCache(cmd)

			// 调用set方法
			err := cc.Set(tc.ctx, tc.biz, tc.phone, tc.code)
			assert.Equal(t, tc.wantErr, err)
		})
	}

}
