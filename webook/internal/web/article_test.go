package web

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"net/http"
	"net/http/httptest"
	"refactor-webook/webook/internal/domain"
	"refactor-webook/webook/internal/service"
	svcmocks "refactor-webook/webook/internal/service/mocks"
	ijwt "refactor-webook/webook/internal/web/jwt"
	"refactor-webook/webook/pkg/logger"
	"testing"
)

func TestArticleHandler_Publish(t *testing.T) {
	testcases := []struct {
		name     string
		mock     func(ctrl *gomock.Controller) service.ArticleService
		reqBody  string
		wantCode int
		wantRes  Result
	}{
		{
			name: "新建并发表成功",
			mock: func(ctrl *gomock.Controller) service.ArticleService {
				articleSvc := svcmocks.NewMockArticleService(ctrl)
				articleSvc.EXPECT().Publish(gomock.Any(), domain.Article{
					Title:   "测试标题",
					Content: "测试内容",
					Author: domain.Author{
						Id: 123,
					},
				}).Return(int64(1), nil)
				return articleSvc
			},
			reqBody:  `{"title":"测试标题", "content":"测试内容"}`,
			wantCode: http.StatusOK,
			wantRes: Result{
				// note 数字转成 any 类型时，返回的是 float64 类型
				Data: float64(1),
			},
		},
		{
			name: "已有帖子发表成功",
			mock: func(ctrl *gomock.Controller) service.ArticleService {
				articleSvc := svcmocks.NewMockArticleService(ctrl)
				articleSvc.EXPECT().Publish(gomock.Any(), domain.Article{
					Id:      1,
					Title:   "测试标题",
					Content: "测试内容",
					Author: domain.Author{
						Id: 123,
					},
				}).Return(int64(1), nil)
				return articleSvc
			},
			reqBody:  `{"id":1, "title":"测试标题", "content":"测试内容"}`,
			wantCode: http.StatusOK,
			wantRes: Result{
				// note 数字转成 any 类型时，返回的是 float64 类型
				Data: float64(1),
			},
		},
		{
			name: "发表失败",
			mock: func(ctrl *gomock.Controller) service.ArticleService {
				articleSvc := svcmocks.NewMockArticleService(ctrl)
				articleSvc.EXPECT().Publish(gomock.Any(), domain.Article{
					Id:      1,
					Title:   "测试标题",
					Content: "测试内容",
					Author: domain.Author{
						Id: 123,
					},
				}).Return(int64(0), errors.New("mock error"))
				return articleSvc
			},
			reqBody:  `{"id":1, "title":"测试标题", "content":"测试内容"}`,
			wantCode: http.StatusOK,
			wantRes: Result{
				Code: 5,
				Msg:  "系统错误",
			},
		},
		{
			name: "Bind错误",
			mock: func(ctrl *gomock.Controller) service.ArticleService {
				articleSvc := svcmocks.NewMockArticleService(ctrl)
				return articleSvc
			},
			reqBody:  `{"title":"测试标题", "content":"测试内"fdsfd}`,
			wantCode: http.StatusBadRequest,
			// note 因为 handler 中处理 bind 错误的方式是不返回 result
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// 利用mock构造svc
			articleSvc := tc.mock(ctrl)
			// 构造hdl
			hdl := NewArticleHandler(articleSvc, logger.NewNopLogger())
			// 准备服务器和注册路由
			server := gin.Default()
			// note 设置登录态
			server.Use(func(ctx *gin.Context) {
				ctx.Set("user", ijwt.UserClaims{
					Uid: 123,
				})
			})
			hdl.RegisterRoutes(server)

			// 准备Req和记录的 recorder
			req, err := http.NewRequest(http.MethodPost,
				"/articles/publish", bytes.NewReader([]byte(tc.reqBody)))
			assert.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			// 本地接收http请求
			server.ServeHTTP(recorder, req)

			assert.Equal(t, tc.wantCode, recorder.Code)
			// note 因为 handler 中处理 bind 错误的方式是不返回 result
			if recorder.Code != http.StatusOK {
				return
			}
			// 对res反序列化
			var res Result
			err = json.NewDecoder(recorder.Body).Decode(&res)
			assert.NoError(t, err)

			// 断言结果
			assert.Equal(t, tc.wantRes, res)
		})
	}
}
