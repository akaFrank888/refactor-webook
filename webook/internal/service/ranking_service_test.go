package service

import (
	"context"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"refactor-webook/webook/internal/domain"
	svcmocks "refactor-webook/webook/internal/service/mocks"
	"testing"
	"time"
)

func TestBatchRankingService_topN(t *testing.T) {
	now := time.Now()

	testCases := []struct {
		name    string
		mock    func(ctrl *gomock.Controller) (ArticleService, InteractiveService)
		wantRes []domain.Article
		wantErr error
	}{
		{
			name: "正常",
			mock: func(ctrl *gomock.Controller) (ArticleService, InteractiveService) {
				articleSvc := svcmocks.NewMockArticleService(ctrl)
				interactiveSvc := svcmocks.NewMockInteractiveService(ctrl)
				// 模拟第一批 article
				articleSvc.EXPECT().ListPub(gomock.Any(), gomock.Any(), 0, 2).
					Return([]domain.Article{
						{Id: 1, Utime: now},
						{Id: 2, Utime: now},
					}, nil)
				// 获取点赞数
				interactiveSvc.EXPECT().GetByIds(gomock.Any(), "article", []int64{1, 2}).
					Return(map[int64]domain.Interactive{
						1: {LikeCnt: 1},
						2: {LikeCnt: 2},
					}, nil)
				// 模拟第二批 article
				articleSvc.EXPECT().ListPub(gomock.Any(), gomock.Any(), 2, 2).
					Return([]domain.Article{
						{Id: 3, Utime: now},
						{Id: 4, Utime: now},
					}, nil)
				// 获取点赞数
				interactiveSvc.EXPECT().GetByIds(gomock.Any(), "article", []int64{3, 4}).
					Return(map[int64]domain.Interactive{
						3: {LikeCnt: 3},
						4: {LikeCnt: 4},
					}, nil)
				// 没数据了：模拟第三批 article
				articleSvc.EXPECT().ListPub(gomock.Any(), gomock.Any(), 4, 2).
					Return([]domain.Article{}, nil)

				return articleSvc, interactiveSvc
			},
			wantRes: []domain.Article{
				{Id: 4, Utime: now},
				{Id: 3, Utime: now},
				{Id: 2, Utime: now},
			},
			wantErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			articleSvc, interSvc := tc.mock(ctrl)
			// 因为要创建方便测试的 svc，所以不调用 new 来创建，而是直接 &BatchRankingService{}
			svc := &BatchRankingService{
				articleSvc: articleSvc,
				interSvc:   interSvc,
				batchSize:  2,
				n:          3,
				scoreFunc: func(likeCnt int64, utime time.Time) float64 {
					// 简化处理：直接返回 likeCnt
					return float64(likeCnt)
				},
			}
			articles, err := svc.topN(context.Background())
			assert.Equal(t, tc.wantErr, err)
			assert.Equal(t, tc.wantRes, articles)
		})
	}
}
