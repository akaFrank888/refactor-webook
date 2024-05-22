package repository

import (
	"context"
	"refactor-webook/webook/internal/domain"
)

type ArticleReaderRepository interface {
	// Save note 线上库中，不能用 id 来区分是新建还是更新（因为总会携带 id ），所以命名为Save()，在其实现细节中完成对新建和更新的区分
	Save(ctx context.Context, article domain.Article) error
}
