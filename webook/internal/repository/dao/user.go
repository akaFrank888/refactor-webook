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
	ErrUserDuplicateUser = errors.New("邮箱/手机号冲突")
)

type UserDao interface {
	Insert(ctx context.Context, user User) error
	FindByEmail(ctx context.Context, email string) (User, error)
	FindById(ctx context.Context, id int64) (User, error)
	UpdateById(ctx context.Context, user User) error
	FindByPhone(ctx context.Context, phone string) (User, error)
	FindByOpenId(ctx context.Context, openId string) (User, error)
}

func (dao *GormUserDao) FindById(ctx context.Context, id int64) (User, error) {
	user := User{}
	return user, dao.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
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
			return ErrUserDuplicateUser
		}
	}
	return err
}

func (dao *GormUserDao) FindByEmail(ctx context.Context, email string) (User, error) {
	var user User
	err := dao.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	return user, err
}

func (dao *GormUserDao) UpdateById(ctx context.Context, user User) error {
	res := dao.db.WithContext(ctx).Model(&user).Where("id=?", user.Id).Updates(map[string]any{
		"utime":    time.Now().UnixMilli(), // 更新时间
		"nickname": user.Nickname,
		"birthday": user.Birthday,
		"resume":   user.Resume,
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("id错误，更新失败")
	}
	return nil
}

func (dao *GormUserDao) FindByPhone(ctx context.Context, phone string) (User, error) {
	var user User
	err := dao.db.WithContext(ctx).Where("phone = ?", phone).First(&user).Error

	return user, err
}

func (dao *GormUserDao) FindByOpenId(ctx context.Context, openId string) (User, error) {
	var user User
	err := dao.db.WithContext(ctx).Where("wechat_open_id = ?", openId).First(&user).Error

	return user, err
}

type User struct {
	Id int64 `gorm:"primary_key, autoIncrement"`
	// note 可为null，因为可以不用email注册，而用手机号
	Email    sql.NullString `gorm:"unique"`
	Password string
	Phone    sql.NullString `gorm:"unique"`

	// 创建时间  避免时区问题，一律用 UTC 0 的毫秒数【若要转成符合中国的时区，要么让前端处理，要么在web层给前端的时候转成UTC 8 的时区】
	Ctime int64
	Utime int64

	Nickname string `gorm:"type=varchar(20)"`
	Birthday int64
	Resume   string `gorm:"type=varchar(200)"`

	// note
	// 1 如果查询要求同时使用 openid 和 unionid，就要创建联合唯一索引
	// 2 如果查询只用 openid，那么就在 openid 上创建唯一索引，或者 <openid, unionId> 联合索引
	// 3 如果查询只用 unionid，那么就在 unionid 上创建唯一索引，或者 <unionid, openid> 联合索引
	WechatOpenId  sql.NullString `gorm:"unique"`
	WechatUnionId sql.NullString
}
