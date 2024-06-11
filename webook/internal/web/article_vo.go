package web

type ArticleVo struct {
	Id    int64  `json:"id,omitempty"`
	Title string `json:"title,omitempty"`

	Abstract   string `json:"abstract,omitempty"`
	Content    string `json:"content,omitempty"`
	AuthorId   int64  `json:"authorId,omitempty"`
	AuthorName string `json:"authorId,omitempty"`

	Ctime string `json:"ctime,omitempty"`
	Utime string `json:"utime,omitempty"`

	Status uint8 `json:"status,omitempty"`
}
