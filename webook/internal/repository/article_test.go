package repository

import (
	"context"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"refactor-webook/webook/internal/domain"
	"refactor-webook/webook/internal/repository/dao"
	daomocks "refactor-webook/webook/internal/repository/dao/mocks"
	"testing"
)

func TestCachedArticleRepository_SyncV1(t *testing.T) {
	testCases := []struct {
		name string
		mock func(ctrl *gomock.Controller) (dao.ArticleAuthorDao, dao.ArticleReaderDao)

		article domain.Article

		wantId  int64
		wantErr error
	}{
		{
			name: "新建并同步成功",
			mock: func(ctrl *gomock.Controller) (dao.ArticleAuthorDao, dao.ArticleReaderDao) {
				authorDao := daomocks.NewMockArticleAuthorDao(ctrl)
				readerDao := daomocks.NewMockArticleReaderDao(ctrl)
				authorDao.EXPECT().Create(gomock.Any(), dao.Article{
					Title:    "测试标题",
					Content:  "测试内容",
					AuthorId: 123,
				}).Return(int64(1), nil)
				readerDao.EXPECT().Upsert(gomock.Any(), dao.Article{
					Id:       1,
					Title:    "测试标题",
					Content:  "测试内容",
					AuthorId: 123,
				}).Return(nil)
				return authorDao, readerDao
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
			name: "更新并同步成功",
			mock: func(ctrl *gomock.Controller) (dao.ArticleAuthorDao, dao.ArticleReaderDao) {
				authorDao := daomocks.NewMockArticleAuthorDao(ctrl)
				readerDao := daomocks.NewMockArticleReaderDao(ctrl)
				authorDao.EXPECT().Update(gomock.Any(), dao.Article{
					Id:       1,
					Title:    "测试标题",
					Content:  "测试内容",
					AuthorId: 123,
				}).Return(nil)
				readerDao.EXPECT().Upsert(gomock.Any(), dao.Article{
					Id:       1,
					Title:    "测试标题",
					Content:  "测试内容",
					AuthorId: 123,
				}).Return(nil)
				return authorDao, readerDao
			},
			article: domain.Article{
				Id:      1,
				Title:   "测试标题",
				Content: "测试内容",
				Author: domain.Author{
					Id: 123,
				},
			},
			wantId:  1,
			wantErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			authorDao, readerDao := tc.mock(ctrl)
			repo := NewArticleRepositoryV2(authorDao, readerDao)
			id, err := repo.SyncV1(context.Background(), tc.article)
			assert.Equal(t, tc.wantErr, err)
			assert.Equal(t, tc.wantId, id)
		})
	}
}
