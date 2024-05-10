package dao

import (
	"context"
	"database/sql"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"testing"
)

func TestGormUserDao_Insert(t *testing.T) {
	testCases := []struct {
		name string
		mock func(t *testing.T) *sql.DB // note 所有的ORM框架都是建立在sql*DB基础上
		ctx  context.Context
		user User

		wantErr error
	}{
		{
			name: "插入成功",
			mock: func(t *testing.T) *sql.DB {
				db, mock, err := sqlmock.New()
				assert.NoError(t, err)
				mockRes := sqlmock.NewResult(123, 1)
				// 这边要求传入的是 sql 的正则表达式
				mock.ExpectExec("INSERT INTO .*").
					WillReturnResult(mockRes)
				return db
			},
			ctx: context.Background(),
			user: User{
				Nickname: "Tom",
			},
		},
		{
			name: "邮箱冲突",
			mock: func(t *testing.T) *sql.DB {
				db, mock, err := sqlmock.New()
				assert.NoError(t, err)
				// 这边要求传入的是 sql 的正则表达式
				mock.ExpectExec("INSERT INTO .*").
					WillReturnError(&mysqlDriver.MySQLError{Number: 1062})
				return db
			},
			ctx: context.Background(),
			user: User{
				Nickname: "Tom",
			},
			wantErr: ErrUserDuplicateUser,
		},
		{
			name: "db错误",
			mock: func(t *testing.T) *sql.DB {
				db, mock, err := sqlmock.New()
				assert.NoError(t, err)
				// 这边要求传入的是 sql 的正则表达式
				mock.ExpectExec("INSERT INTO .*").
					WillReturnError(errors.New("db错误"))
				return db
			},
			ctx: context.Background(),
			user: User{
				Nickname: "Tom",
			},
			wantErr: errors.New("db错误"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sqlDB := tc.mock(t)
			db, err := gorm.Open(mysql.New(mysql.Config{
				Conn:                      sqlDB,
				SkipInitializeWithVersion: true,
			}), &gorm.Config{
				DisableAutomaticPing:   true,
				SkipDefaultTransaction: true,
			})
			assert.NoError(t, err)
			dao := NewUserDao(db)
			err = dao.Insert(tc.ctx, tc.user)
			assert.Equal(t, tc.wantErr, err)
		})
	}
}
