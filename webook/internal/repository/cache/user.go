package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"refactor-webook/webook/internal/domain"
	"time"
)

var ErrKeyNotExist = redis.Nil

type UserCache interface {
	Get(ctx context.Context, id int64) (domain.User, error)
	Set(ctx context.Context, user domain.User) error
	Del(ctx context.Context, id int64) error
}

// RedisUserCache note 用Redis实现的Cache，后面可继续实现基于内存等其他的Cache的具体实现
type RedisUserCache struct {
	cmd        redis.Cmdable
	expiration time.Duration
}

func NewUserCache(cmd redis.Cmdable) UserCache {
	return &RedisUserCache{
		cmd:        cmd,
		expiration: time.Minute * 15, // note 向repo层屏蔽过期时间设置问题（让repo层不再关心过期时间设置问题）
	}
}

// key首字母小写，是内部方法（不暴露在接口中）
func (uc *RedisUserCache) key(id int64) string {
	return fmt.Sprintf("user:info:%d", id) // note 向repo层屏蔽key的组成
}

func (uc *RedisUserCache) Set(ctx context.Context, user domain.User) error {
	key := uc.key(user.Id)
	val, err := json.Marshal(user) // note 向repo层屏蔽序列化过程
	if err != nil {
		return err
	}
	return uc.cmd.Set(ctx, key, val, uc.expiration).Err()
}

func (uc *RedisUserCache) Get(ctx context.Context, id int64) (domain.User, error) {
	key := uc.key(id)
	val, err := uc.cmd.Get(ctx, key).Result() // note 要加Result()
	if err != nil {
		return domain.User{}, err
	}
	user := domain.User{}
	err = json.Unmarshal([]byte(val), &user)
	return user, err
}

func (uc *RedisUserCache) Del(ctx context.Context, id int64) error {
	return uc.cmd.Del(ctx, uc.key(id)).Err()
}

/*

// UserCacheV1 避免下面这种设计，原因是：【面向接口编程】
type UserCacheV1 struct {
	client     *redis.Client // note redis.Client是结构体，是redis.Cmdable的其中一种实现（还有集群redis等其他实现）。而redis.Cmdable是接口，更符合面向接口编程
	expiration time.Duration
}

// NewUserCacheV1 避免下面这种设计，原因是：良好的设计是不要自己去初始化需要的东西，而是外面传进来【依赖注入】
func NewUserCacheV1(addr string) *UserCache {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return &UserCache{
		cmd:        client, // 将具体实现的client赋值给接口类型的cmd
		expiration: time.Minute * 15,
	}
}

*/
