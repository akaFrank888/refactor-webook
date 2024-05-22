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
}

type Author struct {
	Id   int64
	Name string
}
