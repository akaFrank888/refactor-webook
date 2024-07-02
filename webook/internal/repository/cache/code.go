package cache

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	lru "github.com/hashicorp/golang-lru"
	"github.com/redis/go-redis/v9"
	"sync"
	"time"
)

var (
	//go:embed lua/set_code.lua
	luaSetCode string
	//go:embed lua/verify_code.lua
	luaVerifyCode string

	ErrCodeSendTooMany   = errors.New("发送太频繁")
	ErrCodeVerifyTooMany = errors.New("验证太频繁")
)

type CodeCache interface {
	Set(ctx context.Context, biz, phone, code string) error
	Verify(ctx context.Context, biz, phone, code string) (bool, error)
}

// LocalCodeCache 本地缓存实现
type LocalCodeCache struct {
	cache      *lru.Cache
	lock       sync.Mutex
	expiration time.Duration
}

func NewLocalCodeCache(c *lru.Cache, expiration time.Duration) *LocalCodeCache {
	return &LocalCodeCache{
		cache:      c,
		expiration: expiration,
	}
}

func (l *LocalCodeCache) Set(ctx context.Context, biz string, phone string, code string) error {

	l.lock.Lock()
	defer l.lock.Unlock()
	// 这里可以考虑用读写锁来优化，但是效果不会很好
	// 因为你可以预期，大部分时候是要走到写锁里面的
	key := l.key(biz, phone)

	now := time.Now()
	val, ok := l.cache.Get(key)
	if !ok {
		// 说明没有验证码
		l.cache.Add(key, codeItem{
			code:   code,
			cnt:    3,
			expire: now.Add(l.expiration),
		})
		return nil
	}
	itm, ok := val.(codeItem)
	if !ok {
		// 理论上来说这是不可能的
		return errors.New("系统错误")
	}
	if itm.expire.Sub(now) > time.Minute*9 {
		// 不到一分钟
		return ErrCodeSendTooMany
	}
	// 重发
	l.cache.Add(key, codeItem{
		code:   code,
		cnt:    3,
		expire: now.Add(l.expiration),
	})
	return nil
}

func (l *LocalCodeCache) Verify(ctx context.Context, biz string, phone string, inputCode string) (bool, error) {
	l.lock.Lock()
	defer l.lock.Unlock()
	key := l.key(biz, phone)
	val, ok := l.cache.Get(key)
	if !ok {
		// 都没发验证码
		return false, ErrKeyNotExist
	}
	itm, ok := val.(codeItem)
	if !ok {
		// 理论上来说这是不可能的
		return false, errors.New("系统错误")
	}
	if itm.cnt <= 0 {
		return false, ErrCodeVerifyTooMany
	}
	itm.cnt--
	return itm.code == inputCode, nil
}

func (l *LocalCodeCache) key(biz string, phone string) string {
	return fmt.Sprintf("phone_code:%s:%s", biz, phone)
}

type codeItem struct {
	code string
	// 可验证次数
	cnt int
	// 过期时间
	expire time.Time
}

type RedisCodeCache struct {
	cmd redis.Cmdable
}

func (c *RedisCodeCache) Set(ctx context.Context, biz, phone, code string) error {
	// 调用rua脚本
	res, err := c.cmd.Eval(ctx, luaSetCode, []string{c.key(biz, phone)}, code).Int()
	if err != nil {
		// 调用redis失败
		return err
	}
	switch res {
	case -2:
		return errors.New("验证码存在，但是没有过期时间")
	case -1:
		return ErrCodeSendTooMany
	default:
		return nil
	}
}

func (c *RedisCodeCache) Verify(ctx context.Context, biz, phone, code string) (bool, error) {
	res, err := c.cmd.Eval(ctx, luaVerifyCode, []string{c.key(biz, phone)}, code).Int()
	if err != nil {
		// 调用 redis 出了问题
		return false, err
	}
	switch res {
	case -2:
		return false, nil
	case -1:
		return false, ErrCodeVerifyTooMany
	default:
		return true, nil
	}
}

func (c *RedisCodeCache) key(biz, phone string) string {
	return fmt.Sprintf("phone_code:%s:%s", biz, phone)
}

func NewCodeCache(cmd redis.Cmdable) CodeCache {
	return &RedisCodeCache{
		cmd: cmd,
	}
}
