package domain

import "time"

type Article struct {
	Id      int64
	Title   string
	Content string
	// note 用户领域的 User ，在帖子领域就变成了 “值对象” 【DDD原则】
	Author Author

	Ctime time.Time
	Utime time.Time

	Status ArticleStatus
}

// Abstract 取content的前128个字作为摘要
func (a *Article) Abstract() string {
	str := []rune(a.Content)
	if len(str) > 128 {
		str = str[:128]
	}
	return string(str)

}

// AbstractV1 用 gpt 生成 abstract
func (a *Article) AbstractV1() string {

}

type Author struct {
	Id   int64
	Name string
}

// ArticleStatus 0-255的无符号整数表示状态
type ArticleStatus uint8

func (as ArticleStatus) ToUint8() uint8 {
	return uint8(as)
}

const (
	// ArticleStatusUnknown 这是一个未知状态
	ArticleStatusUnknown ArticleStatus = iota
	// ArticleStatusUnpublished 未发表
	ArticleStatusUnpublished
	// ArticleStatusPublished 已发表
	ArticleStatusPublished
	// ArticleStatusPrivate 仅自己可见
	ArticleStatusPrivate
)
