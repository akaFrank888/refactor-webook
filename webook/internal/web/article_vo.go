package web

type ArticleVo struct {
	Id    int64  `json:"id,omitempty"`
	Title string `json:"title,omitempty"`

	Abstract   string `json:"abstract,omitempty"`
	Content    string `json:"content,omitempty"`
	AuthorId   int64  `json:"authorId,omitempty"`
	AuthorName string `json:"authorName,omitempty"`

	// note 常见需求：阅读、点赞数、收藏数和是否赞过、是否收藏
	ReadCnt    int64 `json:"readCnt"`
	LikeCnt    int64 `json:"likeCnt"`
	CollectCnt int64 `json:"collectCnt"`
	Liked      bool  `json:"liked"`
	Collected  bool  `json:"collected"`

	Ctime string `json:"ctime,omitempty"`
	Utime string `json:"utime,omitempty"`

	Status uint8 `json:"status,omitempty"`
}
