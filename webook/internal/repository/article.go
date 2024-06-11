package repository

import (
	"context"
	"gorm.io/gorm"
	"refactor-webook/webook/internal/domain"
	"refactor-webook/webook/internal/repository/cache"
	"refactor-webook/webook/internal/repository/dao"
	"refactor-webook/webook/pkg/kit"
	"refactor-webook/webook/pkg/logger"
	"time"
)

type ArticleRepository interface {
	Create(ctx context.Context, article domain.Article) (int64, error)
	Update(ctx context.Context, article domain.Article) error
	Sync(ctx context.Context, article domain.Article) (int64, error)
	SyncStatus(ctx context.Context, uid int64, id int64, private domain.ArticleStatus) error
	GetByAuthor(ctx context.Context, uid int64, offset int, limit int) ([]domain.Article, error)
	GetById(ctx context.Context, id int64) (domain.Article, error)

	// 读者
	GetPubById(ctx context.Context, id int64) (domain.Article, error)
}

type CachedArticleRepository struct {
	dao      dao.ArticleDao
	cache    cache.ArticleCache
	l        logger.LoggerV1
	userRepo UserRepository

	// V2 写法专用  ————  在repo层同步制作库和线上库的数据（完成分发） 【不在repo开启事务】
	authorDao dao.ArticleAuthorDao
	readerDao dao.ArticleReaderDao

	// 【在repo开启事务】
	db *gorm.DB
}

func (repo *CachedArticleRepository) SyncStatus(ctx context.Context, uid int64, id int64, status domain.ArticleStatus) error {

	err := repo.dao.SyncStatus(ctx, uid, id, status.ToUint8())
	if err == nil {
		// note 在新写文章、编辑文章、发表的时候都要清除相关的缓存【保证缓存的一致性】
		err = repo.cache.DeleteFirstPage(ctx, uid)
		if err != nil {
			// 清除缓存失败，记录日志
			repo.l.Error("删除首页缓存失败", logger.Error(err))
		}
	}
	return err
}

func NewArticleRepository(dao dao.ArticleDao, cache cache.ArticleCache) ArticleRepository {
	return &CachedArticleRepository{
		dao:   dao,
		cache: cache,
	}
}

func NewArticleRepositoryV2(authorDao dao.ArticleAuthorDao, readerDao dao.ArticleReaderDao) *CachedArticleRepository {
	return &CachedArticleRepository{
		authorDao: authorDao,
		readerDao: readerDao,
	}
}

func (repo *CachedArticleRepository) Create(ctx context.Context, article domain.Article) (int64, error) {
	articleId, err := repo.dao.Insert(ctx, repo.toPersistent(article))
	if err == nil {
		// note 在新写文章、编辑文章、发表的时候都要清除相关的缓存【保证缓存的一致性】
		err = repo.cache.DeleteFirstPage(ctx, article.Author.Id)
		if err != nil {
			// 清除缓存失败，记录日志
			repo.l.Error("删除首页缓存失败", logger.Error(err))
		}
	}
	return articleId, err
}

func (repo *CachedArticleRepository) Update(ctx context.Context, article domain.Article) error {
	err := repo.dao.UpdateById(ctx, repo.toPersistent(article))
	if err == nil {
		// note 在新写文章、编辑文章、发表的时候都要清除相关的缓存【保证缓存的一致性】
		err = repo.cache.DeleteFirstPage(ctx, article.Author.Id)
		if err != nil {
			// 清除缓存失败，记录日志
			repo.l.Error("删除首页缓存失败", logger.Error(err))
		}
	}
	return err
}

// Sync note 【在dao层完成“发表”中制作库和线上库的分发或者叫同步，同库不同表，引入事务】
func (repo *CachedArticleRepository) Sync(ctx context.Context, article domain.Article) (int64, error) {
	articleId, err := repo.dao.Sync(ctx, repo.toPersistent(article))
	if err != nil {
		return 0, err
	}

	// note 在新写文章、编辑文章、发表的时候都要清除相关的缓存【保证缓存的一致性】
	err = repo.cache.DeleteFirstPage(ctx, article.Author.Id)
	if err != nil {
		// 清除缓存失败，记录日志
		repo.l.Error("删除首页缓存失败", logger.Error(err))
	}

	// note 在 publish 成功后，将该文章缓存给读者【此处的expiration可以灵活设置：流量大的大V的expiration大，反之则小】
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		user, er := repo.userRepo.FindById(ctx, article.Author.Id)
		if er != nil {
			return
		}
		article.Author.Name = user.Nickname
		err = repo.cache.SetPub(ctx, article)
	}()

	return articleId, err
}

// SyncV1 note 【在repo层完成“发表”中制作库和线上库的分发或者叫同步】【V1：假定制作库和线上库是不同一个数据库，所以是不存在事务】
func (repo *CachedArticleRepository) SyncV1(ctx context.Context, article domain.Article) (int64, error) {
	article2Persistent := repo.toPersistent(article)
	var (
		err error
		id  = article.Id
	)
	if article.Id > 0 {
		err = repo.authorDao.Update(ctx, article2Persistent)
	} else {
		id, err = repo.authorDao.Create(ctx, article2Persistent)
	}
	if err != nil {
		return 0, err
	}

	// note 在新写文章、编辑文章、发表的时候都要清除相关的缓存【保证缓存的一致性】
	err = repo.cache.DeleteFirstPage(ctx, article.Author.Id)
	if err != nil {
		// 清除缓存失败，记录日志
		repo.l.Error("删除首页缓存失败", logger.Error(err))
	}

	// note 这句其实可以省略，因为dao层的gorm会创建个id并赋值给article，但防止dao换了别的实现，还是建议不省略
	article.Id = id
	return id, repo.readerDao.Upsert(ctx, article2Persistent)
}

// SyncV2 note 【在repo层完成“发表”中制作库和线上库的分发或者叫同步】【V2：假定制作库和线上库是同库不同表，且在repo层面完成事务，所以要在repo中跨层调用db】
func (repo *CachedArticleRepository) SyncV2(ctx context.Context, article domain.Article) (int64, error) {
	tx := repo.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return 0, tx.Error
	}
	// 防止后面的业务panic
	defer tx.Rollback()
	// note 用该 tx 封装一个基于该事务的 dao
	authorDao := dao.NewArticleGormAuthorDao(tx)
	readerDao := dao.NewArticleReaderDao(tx)

	article2Persistent := repo.toPersistent(article)
	var (
		err error
		id  = article.Id
	)
	if article.Id > 0 {
		err = authorDao.Update(ctx, article2Persistent)
	} else {
		id, err = authorDao.Create(ctx, article2Persistent)
	}
	if err != nil {
		return 0, err
	}

	// note 在新写文章、编辑文章、发表的时候都要清除相关的缓存【保证缓存的一致性】
	err = repo.cache.DeleteFirstPage(ctx, article.Author.Id)
	if err != nil {
		// 清除缓存失败，记录日志
		repo.l.Error("删除首页缓存失败", logger.Error(err))
	}

	article.Id = id
	err = readerDao.UpsertV2(ctx, article2Persistent)
	if err != nil {
		return 0, err
	}
	// note 手动 commit 要注意：1）“return 0, err”前都会回滚 2）虽然是 commit 后再 rollback 会报错，但不理它即可
	tx.Commit()
	return id, nil

}

func (repo *CachedArticleRepository) GetByAuthor(ctx context.Context, uid int64, offset int, limit int) ([]domain.Article, error) {
	// note 缓存第一页的分页结果，所以要限定offset和limit【场景：用户一般只看第一页就找到结果】
	// 可优化： limit <= 100 也可以读缓存
	if offset == 0 && limit == 100 {
		res, err := repo.cache.GetFirstPage(ctx, uid)
		if err == nil {
			// 缓存命中
			return res, nil
		} else {
			// 未命中或者连接redis的网络问题
			// todo 此处需要日志或监控
		}
	}

	articles, err := repo.dao.GetByAuthor(ctx, uid, offset, limit)
	if err != nil {
		return nil, err
	}
	// note 将articles中的每一个dao类型的article转成domain类型的article
	res := kit.Map[dao.Article, domain.Article](articles, func(idx int, article dao.Article) domain.Article {
		return repo.toDomain(article)
	})

	go func() {
		// note 异步的话，要重新建立一个 context
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		// 回写缓存
		if offset == 0 && limit == 100 {
			err = repo.cache.SetFirstPage(ctx, uid, res)
			if err != nil {
				// 此处需要监控，网络连接问题，需要人工干预。
			}
		}
	}()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		// note 缓存 id=1 的 article 【场景：当用户查询完一个作者的文章list后，大概率会查询作者 id=1 的文章】
		repo.preCache(ctx, res)
	}()
	return res, nil

}

func (repo *CachedArticleRepository) GetById(ctx context.Context, id int64) (domain.Article, error) {
	articleCached, err := repo.cache.Get(ctx, id)
	if err == nil {
		// 缓存命中
		return articleCached, nil
	}

	article, err := repo.dao.GetById(ctx, id)
	if err != nil {
		return domain.Article{}, err
	}

	go func() {
		// 回写缓存
		er := repo.cache.Set(ctx, repo.toDomain(article))
		if er != nil {
			// 日志
		}
	}()

	return repo.toDomain(article), nil
}

// note 需要封装 userRepo 里的 name
func (repo *CachedArticleRepository) GetPubById(ctx context.Context, id int64) (domain.Article, error) {
	articleCached, err := repo.cache.GetPub(ctx, id)
	if err == nil {
		return articleCached, nil
	}
	article, err := repo.dao.GetPubById(ctx, id)
	if err != nil {
		return domain.Article{}, err
	}
	res := repo.toDomain(dao.Article(article)) // note 偷懒的写法：将dao的PublishedArticle转成了dao的Article（但随着业务变复杂，PublishedArticle与Article字段不同后，需要单独为PublishedArticle==>domain.Article设计一个toDomain()）
	author, err := repo.userRepo.FindById(ctx, article.AuthorId)
	if err != nil {
		return domain.Article{}, err
	}
	res.Author.Name = author.Nickname

	// 回写缓存
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		er := repo.cache.SetPub(ctx, res)
		if er != nil {
			// 日志
		}
	}()

	return res, nil

}

func (repo *CachedArticleRepository) toPersistent(article domain.Article) dao.Article {
	return dao.Article{
		// note 因为 dao 层会处理 Ctime 和 Utime ，所以不用传
		Id:       article.Id,
		Title:    article.Title,
		Content:  article.Content,
		AuthorId: article.Author.Id,
		Status:   article.Status.ToUint8(),
	}
}

func (repo *CachedArticleRepository) toDomain(article dao.Article) domain.Article {
	return domain.Article{
		Id:      article.Id,
		Title:   article.Title,
		Content: article.Content,
		Status:  domain.ArticleStatus(article.Status),
		Author: domain.Author{
			Id: article.AuthorId,
		},
		Ctime: time.UnixMilli(article.Ctime),
		Utime: time.UnixMilli(article.Utime),
	}
}

// 预加载 id = 1 的article
// note 优化：只缓存小文章（考虑Redis内存和缓存性能的平衡）
func (repo *CachedArticleRepository) preCache(ctx context.Context, res []domain.Article) {
	// 1MB
	const contentSizeThrehold = 1024 * 1024

	if len(res) == 0 || len(res[0].Content) > contentSizeThrehold {
		return
	}
	err := repo.cache.Set(ctx, res[0])
	if err != nil {
		repo.l.Error("提前缓存 id=1 的article失败", logger.Error(err))
	}
}
