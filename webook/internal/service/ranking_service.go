package service

import (
	"context"
	"errors"
	"github.com/ecodeclub/ekit/queue"
	"math"
	"refactor-webook/webook/internal/domain"
	"refactor-webook/webook/internal/repository"
	"time"
)

type RankingService interface {
	// TopN “为什么不返回 []article？”  因为要将 topN 的 article 存入缓存中
	TopN(ctx context.Context) error
	GetTopN(ctx context.Context) ([]domain.Article, error)
}

type BatchRankingService struct {
	articleSvc ArticleService
	interSvc   InteractiveService

	batchSize int
	// top n （如果经常变动，就作为方法的参数，否则就作为字段）
	n int64
	// 计算 score
	scoreFunc func(likeCnt int64, utime time.Time) float64

	repo repository.RankingRepository
}

func NewBatchRankingService(articleSvc ArticleService, interSvc InteractiveService) RankingService {
	return &BatchRankingService{
		articleSvc: articleSvc,
		interSvc:   interSvc,
		batchSize:  100,
		n:          100,
		scoreFunc: func(likeCnt int64, utime time.Time) float64 {
			duration := time.Since(utime).Seconds()               // 取秒取分钟都可以
			return float64(likeCnt-1) / math.Pow(duration+2, 1.5) // 对于除法操作，只要有一个操作数是 float 类型，除法的结果不会是整数
		},
	}
}

func (b *BatchRankingService) TopN(ctx context.Context) error {
	articles, err := b.topN(ctx)
	if err != nil {
		return err
	}

	// 最后将 articles 存入缓存中
	return b.repo.ReplaceTopN(ctx, articles)
}

// 因为仅想通过单元测试测试一下榜单计算的算法，而原 TopN 会存入缓存不方便测试，所以就写一个 topN 来只执行榜单算法
func (b *BatchRankingService) topN(ctx context.Context) ([]domain.Article, error) {

	var offset = 0
	ddl := time.Now()
	// note（优化）规定：热榜只考虑更新时间 7 天内的文章
	timeBefore := ddl.Add(-7 * 24 * time.Hour)
	// 创建一个新的结构体用于存入小顶堆
	type Element struct {
		score   float64
		article domain.Article
	}
	// note golang 标准库中没有像 Java 一样的 PriorityQueue，所以只能自己实现或调用开源库的
	// 参考：github.com/ecodeclub/ekit/queue
	minHeap := queue.NewPriorityQueue[Element](int(b.n), func(src Element, dst Element) int {
		if src.score > dst.score {
			return 1
		} else if src.score == dst.score {
			return 0
		} else {
			return -1
		}
	})

	for {
		articles, err := b.articleSvc.ListPub(ctx, ddl, offset, b.batchSize)
		if err != nil {
			return nil, err
		}
		if len(articles) == 0 {
			break
		}
		// 创建切片，保存 id
		ids := make([]int64, len(articles))
		for i, article := range articles {
			ids[i] = article.Id
		}
		// 用 article 的 id 作为 bizId 取点赞数
		interMap, err := b.interSvc.GetByIds(ctx, "article", ids)
		if err != nil {
			return nil, err
		}
		for _, article := range articles {
			utime := article.Utime
			likeCnt := interMap[article.Id].LikeCnt
			score := b.scoreFunc(likeCnt, utime)

			ele := Element{
				score:   score,
				article: article,
			}
			err = minHeap.Enqueue(ele)
			if errors.Is(err, queue.ErrOutOfCapacity) {
				// 最小堆满了
				val, _ := minHeap.Peek() // 忽略 err 因为该 err 是堆为空，显然我们的堆满了，而不是空
				if ele.score < val.score {
					// 忽略
					continue
				} else {
					// 取出堆顶元素，放入新元素
					_, _ = minHeap.Dequeue()
					_ = minHeap.Enqueue(ele)
				}
			}
		}
		offset += b.batchSize
		// 判断是否还有下一批 || 这一批的最后一篇文章是否已经是 7 天前更新的了（dao层返回来的文章列表是按更新时间倒序的）
		if len(articles) < b.batchSize || articles[len(articles)-1].Utime.Before(timeBefore) {
			break
		}
	}

	// minHeap 中就是 topN 最终结果
	// 从 minHeap 中取到 []domain.Article 中
	// note 先出队的元素存在切片后面（因为是第 N 个）
	res := make([]domain.Article, minHeap.Len())
	for i := minHeap.Len() - 1; i >= 0; i-- {
		ele, _ := minHeap.Dequeue()
		res[i] = ele.article
	}
	return res, nil
}

func (b *BatchRankingService) GetTopN(ctx context.Context) ([]domain.Article, error) {
	return b.repo.GetTopN(ctx)
}
