package dao

import (
	"context"
	"database/sql"
	"errors"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"time"
)

var (
	ErrUserDuplicateEmail = errors.New("邮箱冲突")
	ErrRecordNotFound     = gorm.ErrRecordNotFound
)

type UserDao interface {
	Insert(ctx context.Context, user User) error
	FindByEmail(ctx context.Context, email string) (User, error)
}

type GormUserDao struct {
	db *gorm.DB
}

func NewUserDao(db *gorm.DB) UserDao {
	return &GormUserDao{
		db: db,
	}
}

func (dao *GormUserDao) Insert(ctx context.Context, user User) error {
	// 取当前毫秒数   UnixMilli()实现了time-->int64
	now := time.Now().UnixMilli()
	user.Ctime = now
	user.Utime = now

	err := dao.db.WithContext(ctx).Create(&user).Error
	if me, ok := err.(*mysql.MySQLError); ok {
		const uniqueDuplicateKeyError uint16 = 1062
		if me.Number == uniqueDuplicateKeyError {
			return ErrUserDuplicateEmail
		}
	}
	return err
}

func (dao *GormUserDao) FindByEmail(ctx context.Context, email string) (User, error) {
	var user User
	err := dao.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	return user, err
}

type User struct {
	Id int64 `gorm:"primary_key, autoIncrement"`
	// note 可为null，因为可以不用email注册，而用手机号
	Email    sql.NullString `gorm:"unique"`
	Password string

	// 创建时间  避免时区问题，一律用 UTC 0 的毫秒数【若要转成符合中国的时区，要么让前端处理，要么在web层给前端的时候转成UTC 8 的时区】
	Ctime int64
	Utime int64
}
