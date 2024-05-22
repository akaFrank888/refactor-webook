package cache

import (
	"github.com/redis/go-redis/v9"
	"time"
)

type ArticleCache interface {
}

type RedisArticleCache struct {
	cmd        redis.Cmdable
	expiration time.Duration
}

func NewArticleCache(cmd redis.Cmdable) ArticleCache {
	return &RedisArticleCache{
		cmd:        cmd,
		expiration: time.Minute * 15,
	}
}
