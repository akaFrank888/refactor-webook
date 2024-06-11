package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"refactor-webook/webook/internal/domain"
	"time"
)

type ArticleCache interface {
	GetFirstPage(ctx context.Context, uid int64) ([]domain.Article, error)
	SetFirstPage(ctx context.Context, uid int64, res []domain.Article) error
	DeleteFirstPage(ctx context.Context, uid int64) error
	Get(ctx context.Context, id int64) (domain.Article, error)
	Set(ctx context.Context, article domain.Article) error
	GetPub(ctx context.Context, id int64) (domain.Article, error)
	SetPub(ctx context.Context, article domain.Article) error
}

type RedisArticleCache struct {
	client     redis.Cmdable
	expiration time.Duration
}

func NewArticleCache(client redis.Cmdable) ArticleCache {
	return &RedisArticleCache{
		client:     client,
		expiration: time.Minute * 15,
	}
}

func (r *RedisArticleCache) GetFirstPage(ctx context.Context, uid int64) ([]domain.Article, error) {
	// note 因为 Result() 返回的是 String ，所以要在反序列化 Unmarshal 的时候转成 []byte 字节切片。也可以不调 Result()，直接用 Bytes()
	val, err := r.client.Get(ctx, r.firstPageKey(uid)).Result()
	if err != nil {
		return nil, err
	}
	var res []domain.Article
	err = json.Unmarshal([]byte(val), &res)
	return res, err
}

func (r *RedisArticleCache) SetFirstPage(ctx context.Context, uid int64, articles []domain.Article) error {
	// 缓存第一页时，不需要缓存 content，将其替换成 abstract
	for _, article := range articles {
		article.Content = article.Abstract()
	}

	res, err := json.Marshal(articles)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, r.firstPageKey(uid), res, r.expiration).Err()
}

func (r *RedisArticleCache) DeleteFirstPage(ctx context.Context, uid int64) error {
	return r.client.Del(ctx, r.firstPageKey(uid)).Err()
}

func (r *RedisArticleCache) Get(ctx context.Context, id int64) (domain.Article, error) {
	res, err := r.client.Get(ctx, r.key(id)).Bytes()
	if err != nil {
		return domain.Article{}, err
	}
	var article domain.Article
	err = json.Unmarshal(res, &article)
	return article, err
}

func (r *RedisArticleCache) Set(ctx context.Context, article domain.Article) error {
	data, err := json.Marshal(article)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, r.key(article.Id), data, r.expiration).Err()
}

func (r *RedisArticleCache) GetPub(ctx context.Context, id int64) (domain.Article, error) {
	res, err := r.client.Get(ctx, r.pubKey(id)).Bytes()
	if err != nil {
		return domain.Article{}, err
	}
	var article domain.Article
	err = json.Unmarshal(res, &article)
	return article, err
}

func (r *RedisArticleCache) SetPub(ctx context.Context, article domain.Article) error {
	data, err := json.Marshal(article)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, r.pubKey(article.Id), data, r.expiration).Err()
}

func (r *RedisArticleCache) firstPageKey(uid int64) string {
	return fmt.Sprintf("article:first_page:%d", uid)
}
func (r *RedisArticleCache) pubKey(id int64) string {
	return fmt.Sprintf("article:pub:detail:%d", id)
}
func (r *RedisArticleCache) key(id int64) string {
	return fmt.Sprintf("article:detail:%d", id)
}
