package service

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"refactor-webook/webook/internal/domain"
	"refactor-webook/webook/internal/repository"
	repomocks "refactor-webook/webook/internal/repository/mocks"
	"refactor-webook/webook/pkg/logger"
	"testing"
)

func Test_articleService_Publish(t *testing.T) {
	testCases := []struct {
		name string
		mock func(ctrl *gomock.Controller) (repository.ArticleAuthorRepository, repository.ArticleReaderRepository)

		article domain.Article
		wantId  int64
		wantErr error
	}{
		{
			name: "新建并发表成功（即没有文章id传到制作库）",
			mock: func(ctrl *gomock.Controller) (repository.ArticleAuthorRepository, repository.ArticleReaderRepository) {
				authorRepo := repomocks.NewMockArticleAuthorRepository(ctrl)
				readerRepo := repomocks.NewMockArticleReaderRepository(ctrl)
				authorRepo.EXPECT().Create(gomock.Any(), domain.Article{
					Title:   "测试标题",
					Content: "测试内容",
					Author: domain.Author{
						Id: 123,
					},
				}).Return(int64(1), nil)
				readerRepo.EXPECT().Save(gomock.Any(), domain.Article{
					Id:      1,
					Title:   "测试标题",
					Content: "测试内容",
					Author: domain.Author{
						Id: 123,
					},
				}).Return(nil)
				return authorRepo, readerRepo
			},
			article: domain.Article{
				Title:   "测试标题",
				Content: "测试内容",
				Author: domain.Author{
					Id: 123,
				},
			},
			wantId:  1,
			wantErr: nil,
		},
		{
			name: "修改失败（有文章id传到制作库）",
			mock: func(ctrl *gomock.Controller) (repository.ArticleAuthorRepository, repository.ArticleReaderRepository) {
				authorRepo := repomocks.NewMockArticleAuthorRepository(ctrl)
				readerRepo := repomocks.NewMockArticleReaderRepository(ctrl)
				authorRepo.EXPECT().Update(gomock.Any(), domain.Article{
					Id:      12,
					Title:   "测试标题",
					Content: "测试内容",
					Author: domain.Author{
						Id: 123,
					},
				}).Return(errors.New("mock db error"))
				return authorRepo, readerRepo
			},
			article: domain.Article{
				Id:      12,
				Title:   "测试标题",
				Content: "测试内容",
				Author: domain.Author{
					Id: 123,
				},
			},
			wantErr: errors.New("mock db error"),
		},
		{
			name: "修改成功但发表失败（有文章id传到制作库）,重试成功",
			mock: func(ctrl *gomock.Controller) (repository.ArticleAuthorRepository, repository.ArticleReaderRepository) {
				authorRepo := repomocks.NewMockArticleAuthorRepository(ctrl)
				readerRepo := repomocks.NewMockArticleReaderRepository(ctrl)
				authorRepo.EXPECT().Update(gomock.Any(), domain.Article{
					Id:      12,
					Title:   "测试标题",
					Content: "测试内容",
					Author: domain.Author{
						Id: 123,
					},
				})
				readerRepo.EXPECT().Save(gomock.Any(), domain.Article{
					Id:      12,
					Title:   "测试标题",
					Content: "测试内容",
					Author: domain.Author{
						Id: 123,
					},
				}).Return(errors.New("mock db error"))
				// 第二次重试，成功
				readerRepo.EXPECT().Save(gomock.Any(), domain.Article{
					Id:      12,
					Title:   "测试标题",
					Content: "测试内容",
					Author: domain.Author{
						Id: 123,
					},
				}).Return(nil)
				return authorRepo, readerRepo
			},
			article: domain.Article{
				Id:      12,
				Title:   "测试标题",
				Content: "测试内容",
				Author: domain.Author{
					Id: 123,
				},
			},
			wantId:  12,
			wantErr: nil,
		},
		{
			name: "修改成功但发表失败（有文章id传到制作库）,重试失败",
			mock: func(ctrl *gomock.Controller) (repository.ArticleAuthorRepository, repository.ArticleReaderRepository) {
				authorRepo := repomocks.NewMockArticleAuthorRepository(ctrl)
				readerRepo := repomocks.NewMockArticleReaderRepository(ctrl)
				authorRepo.EXPECT().Update(gomock.Any(), domain.Article{
					Id:      12,
					Title:   "测试标题",
					Content: "测试内容",
					Author: domain.Author{
						Id: 123,
					},
				})
				readerRepo.EXPECT().Save(gomock.Any(), domain.Article{
					Id:      12,
					Title:   "测试标题",
					Content: "测试内容",
					Author: domain.Author{
						Id: 123,
					},
				}).Times(3).Return(errors.New("mock db error")) // note 执行3次
				return authorRepo, readerRepo
			},
			article: domain.Article{
				Id:      12,
				Title:   "测试标题",
				Content: "测试内容",
				Author: domain.Author{
					Id: 123,
				},
			},
			wantId:  12,
			wantErr: errors.New("保存到线上库失败，次数耗尽"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			authorRepo, readerRepo := tc.mock(ctrl)
			articleSvc := NewArticleServiceV1(authorRepo, readerRepo, logger.NewNopLogger())

			id, err := articleSvc.PublishV1(context.Background(), tc.article)
			assert.Equal(t, tc.wantId, id)
			assert.Equal(t, tc.wantErr, err)
		})
	}
}
