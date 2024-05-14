package repository

import (
	"context"
	"database/sql"
	"refactor-webook/webook/internal/domain"
	"refactor-webook/webook/internal/repository/cache"
	"refactor-webook/webook/internal/repository/dao"
	"time"
)

var (
	ErrDuplicateUser = dao.ErrUserDuplicateUser
	ErrUserNotFound  = dao.ErrRecordNotFound
)

type UserRepository interface {
	Create(ctx context.Context, user domain.User) error
	FindByEmail(ctx context.Context, email string) (domain.User, error)
	FindById(ctx context.Context, id int64) (domain.User, error)
	UpdateNonZeroFields(ctx context.Context, user domain.User) error
	FindByPhone(ctx context.Context, phone string) (domain.User, error)
	FindByWechat(ctx context.Context, openId string) (domain.User, error)
}

type CacheDUserRepository struct {
	dao   dao.UserDao
	cache cache.UserCache
}

func NewUserRepository(dao dao.UserDao, cache cache.UserCache) UserRepository {
	return &CacheDUserRepository{
		dao:   dao,
		cache: cache,
	}
}

func (repo *CacheDUserRepository) Create(ctx context.Context, user domain.User) error {
	return repo.dao.Insert(ctx, toPersistent(user))
}

func (repo *CacheDUserRepository) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	user, err := repo.dao.FindByEmail(ctx, email)
	if err != nil {
		return domain.User{}, err
	}
	return toDomain(user), nil
}

// FindById 先从Cache中找，再查dao层的数据库（，再回写缓存）
func (repo *CacheDUserRepository) FindById(ctx context.Context, id int64) (domain.User, error) {
	user, err := repo.cache.Get(ctx, id)
	// note 缓存时返回了error就直接查dao
	if err == nil {
		// cache命中
		return user, nil
	}
	// note err不为nil有多种可能：
	// 1） 缓存中没有key，但redis正常
	// 2） 访问redis有问题。可能是连不上网，也可能redis本身崩了

	// note cache未命中 ==》查dao层，回写缓存
	u, err := repo.dao.FindById(ctx, id)
	user = toDomain(u)

	if err != nil {
		return domain.User{}, err
	}

	/*
		// 也可以异步实现set
		go func() {
			err = repo.cache.Set(ctx, user)
			if err != nil {
				log.Println(err)
			}
		}()
	*/

	// note 回写缓存可以忽略err处理：因为这次没存进缓存，下次直接查数据库就行了。而且接受了err，也只说明连接redis的网络和本身有问题，无法解决。
	_ = repo.cache.Set(ctx, user)

	return user, nil
}

func (repo *CacheDUserRepository) UpdateNonZeroFields(ctx context.Context, user domain.User) error {
	// 先更新DB，后更新cache
	err := repo.dao.UpdateById(ctx, toPersistent(user))
	if err != nil {
		return err
	}
	// note 延迟一秒，忽略err
	time.AfterFunc(time.Second, func() {
		_ = repo.cache.Del(ctx, user.Id)
	})
	return repo.cache.Del(ctx, user.Id)
}

func (repo *CacheDUserRepository) FindByPhone(ctx context.Context, phone string) (domain.User, error) {
	user, err := repo.dao.FindByPhone(ctx, phone)
	if err != nil {
		return domain.User{}, err
	}
	return toDomain(user), nil
}

func (repo *CacheDUserRepository) FindByWechat(ctx context.Context, openId string) (domain.User, error) {
	user, err := repo.dao.FindByOpenId(ctx, openId)
	if err != nil {
		return domain.User{}, err
	}
	return toDomain(user), nil
}

func toPersistent(user domain.User) dao.User {
	return dao.User{
		Id: user.Id,
		Email: sql.NullString{
			String: user.Email,
			Valid:  user.Email != "",
		},
		Password: user.Password,
		Phone: sql.NullString{
			String: user.Phone,
			Valid:  user.Phone != "",
		},
		Nickname: user.Nickname,
		Birthday: user.Birthday.UnixMilli(),
		Resume:   user.Resume,
		WechatUnionId: sql.NullString{
			String: user.WechatInfo.UnionId,
			Valid:  user.WechatInfo.UnionId != "",
		},
		WechatOpenId: sql.NullString{
			String: user.WechatInfo.OpenId,
			Valid:  user.WechatInfo.OpenId != "",
		},
	}

}

func toDomain(user dao.User) domain.User {
	return domain.User{
		Id:       user.Id,
		Email:    user.Email.String,
		Password: user.Password,
		Phone:    user.Phone.String,
		Nickname: user.Nickname,
		// UTC 0的毫秒 -> time
		Birthday: time.UnixMilli(user.Birthday),
		Resume:   user.Resume,
		// UTC 0的毫秒 -> time
		Ctime: time.UnixMilli(user.Ctime),
		WechatInfo: domain.WechatInfo{
			OpenId:  user.WechatOpenId.String,
			UnionId: user.WechatUnionId.String,
		},
	}
}
