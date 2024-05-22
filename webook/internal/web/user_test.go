package web

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"net/http"
	"net/http/httptest"
	"refactor-webook/webook/internal/domain"
	"refactor-webook/webook/internal/service"
	svcmocks "refactor-webook/webook/internal/service/mocks"
	ijwt "refactor-webook/webook/internal/web/jwt"
	"testing"
)

func TestUserHandler_SignUp(t *testing.T) {
	testCases := []struct {
		name string
		mock func(ctrl *gomock.Controller) (service.UserService, service.CodeService, ijwt.Handler)
		// 预期中的输入
		reqBuilder func(t *testing.T) *http.Request
		// 预期中的输出
		wantCode int
		wantBody Result
	}{
		// 定义测试用例
		// note 先测试正常的流程（最长的流程），再考虑异常流程
		{
			name: "注册成功",
			mock: func(ctrl *gomock.Controller) (service.UserService, service.CodeService, ijwt.Handler) {
				userSvc := svcmocks.NewMockUserService(ctrl)
				userSvc.EXPECT().SignUp(gomock.Any(), domain.User{
					Email:    "123@qq.com",
					Password: "hello!123",
				}).Return(nil)
				// note CodeSvc没有用到的话，也可以直接写nil
				// return userSvc, nil
				codeSvc := svcmocks.NewMockCodeService(ctrl)
				return userSvc, codeSvc, nil
			},
			reqBuilder: func(t *testing.T) *http.Request {
				req, err := http.NewRequest(http.MethodPost, "/users/signup", bytes.NewReader(
					[]byte(`{"email": "123@qq.com", "password": "hello!123", "confirmPassword":"hello!123"}`)))
				// note 易漏：为req的header添加content-type（因为Bind方法）
				req.Header.Set("Content-Type", "application/json")
				assert.NoError(t, err)
				return req
			},
			wantCode: http.StatusOK,
			wantBody: Result{
				Msg: "注册成功",
			},
		},
		{
			name: "非JSON输入",
			mock: func(ctrl *gomock.Controller) (service.UserService, service.CodeService, ijwt.Handler) {
				// 还执行不到svc，所以直接设为nil
				return nil, nil, nil
			},
			reqBuilder: func(t *testing.T) *http.Request {
				req, err := http.NewRequest(http.MethodPost, "/users/signup", bytes.NewReader(
					// 构造一个错误的JSON
					[]byte(`{"email":"123@qq.com",}`)))
				req.Header.Set("Content-Type", "application/json")
				assert.NoError(t, err)
				return req
			},
			// Bind()出错，返回400
			wantCode: http.StatusBadRequest,
			wantBody: Result{
				Code: 5,
				Msg:  "系统错误",
			},
		},
		{
			name: "两次密码不一致",
			mock: func(ctrl *gomock.Controller) (service.UserService, service.CodeService, ijwt.Handler) {
				// 还执行不到svc，所以直接设为nil
				return nil, nil, nil
			},
			reqBuilder: func(t *testing.T) *http.Request {
				req, err := http.NewRequest(http.MethodPost, "/users/signup", bytes.NewReader(
					// 构造一个错误的JSON
					[]byte(`{"email": "123@qq.com", "password": "hello!123", "confirmPassword":"hello!"}`)))
				req.Header.Set("Content-Type", "application/json")
				assert.NoError(t, err)
				return req
			},
			// Bind()出错，返回400
			wantCode: http.StatusOK,
			wantBody: Result{
				Code: 4,
				Msg:  "两次密码不一致",
			},
		},
		{
			name: "邮箱格式不正确",
			mock: func(ctrl *gomock.Controller) (service.UserService, service.CodeService, ijwt.Handler) {
				// 还执行不到svc，所以直接设为nil
				return nil, nil, nil
			},
			reqBuilder: func(t *testing.T) *http.Request {
				req, err := http.NewRequest(http.MethodPost, "/users/signup", bytes.NewReader(
					[]byte(`{"email": "123qq.com", "password": "hello!123", "confirmPassword":"hello!123"}`)))
				// note 易漏：为req的header添加content-type（因为Bind方法）
				req.Header.Set("Content-Type", "application/json")
				assert.NoError(t, err)
				return req
			},
			wantCode: http.StatusOK,
			wantBody: Result{
				Code: 4,
				Msg:  "邮箱格式不正确",
			},
		},
		{
			name: "密码格式不正确",
			mock: func(ctrl *gomock.Controller) (service.UserService, service.CodeService, ijwt.Handler) {
				// 还执行不到svc，所以直接设为nil
				return nil, nil, nil
			},
			reqBuilder: func(t *testing.T) *http.Request {
				req, err := http.NewRequest(http.MethodPost, "/users/signup", bytes.NewReader(
					[]byte(`{"email": "123@qq.com", "password": "1", "confirmPassword":"1"}`)))
				// note 易漏：为req的header添加content-type（因为Bind方法）
				req.Header.Set("Content-Type", "application/json")
				assert.NoError(t, err)
				return req
			},
			wantCode: http.StatusOK,
			wantBody: Result{
				Code: 4,
				Msg:  "密码格式不正确，至少包含1个字母、1个数字、1个特殊字符且密码总长度至少为8个字符",
			},
		},
		{
			name: "注册的邮箱已存在",
			mock: func(ctrl *gomock.Controller) (service.UserService, service.CodeService, ijwt.Handler) {
				userSvc := svcmocks.NewMockUserService(ctrl)
				userSvc.EXPECT().SignUp(gomock.Any(), domain.User{
					Email:    "123@qq.com",
					Password: "hello!123",
					// note 要返回邮箱冲突的特定错误
				}).Return(service.ErrDuplicateUser)
				codeSvc := svcmocks.NewMockCodeService(ctrl)
				return userSvc, codeSvc, nil
			},
			reqBuilder: func(t *testing.T) *http.Request {
				req, err := http.NewRequest(http.MethodPost, "/users/signup", bytes.NewReader(
					[]byte(`{"email": "123@qq.com", "password": "hello!123", "confirmPassword":"hello!123"}`)))
				// note 易漏：为req的header添加content-type（因为Bind方法）
				req.Header.Set("Content-Type", "application/json")
				assert.NoError(t, err)
				return req
			},
			wantCode: http.StatusOK,
			wantBody: Result{
				Code: 4,
				Msg:  "邮箱冲突",
			},
		},
		{
			name: "系统错误",
			mock: func(ctrl *gomock.Controller) (service.UserService, service.CodeService, ijwt.Handler) {
				userSvc := svcmocks.NewMockUserService(ctrl)
				userSvc.EXPECT().SignUp(gomock.Any(), domain.User{
					Email:    "123@qq.com",
					Password: "hello!123",
				}).Return(errors.New("模拟系统异常，如db异常"))
				// note CodeSvc没有用到的话，也可以直接写nil
				// return userSvc, nil
				codeSvc := svcmocks.NewMockCodeService(ctrl)
				return userSvc, codeSvc, nil
			},
			reqBuilder: func(t *testing.T) *http.Request {
				req, err := http.NewRequest(http.MethodPost, "/users/signup", bytes.NewReader(
					[]byte(`{"email": "123@qq.com", "password": "hello!123", "confirmPassword":"hello!123"}`)))
				// note 易漏：为req的header添加content-type（因为Bind方法）
				req.Header.Set("Content-Type", "application/json")
				assert.NoError(t, err)
				return req
			},
			wantCode: http.StatusOK,
			wantBody: Result{
				Code: 5,
				Msg:  "系统错误",
			},
		},
	}

	// 执行测试用例
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// 利用mock构造svc
			userSvc, codeSvc, jwtHdl := tc.mock(ctrl)
			// 构造hdl
			hdl := NewUserHandler(userSvc, codeSvc, jwtHdl)
			// 准备服务器和注册路由
			server := gin.Default()
			hdl.RegisterRoutes(server)
			// 准备请求和响应
			req := tc.reqBuilder(t)
			recorder := httptest.NewRecorder()

			// 本地接收http请求
			server.ServeHTTP(recorder, req)

			// 断言结果
			assert.Equal(t, tc.wantCode, recorder.Code)
			// 对res反序列化
			var res Result
			err := json.NewDecoder(recorder.Body).Decode(&res)
			assert.NoError(t, err)
			assert.Equal(t, tc.wantBody, res)

		})
	}
}

func TestEmailPattern(t *testing.T) {
	testCases := []struct {
		// 用例的名字，说请测试的场景
		name string
		// 预期输入
		email string
		// 预期输出
		match bool
	}{
		{
			name:  "不带@",
			email: "123456",
			match: false,
		},
		{
			name:  "带@ 但是没后缀",
			email: "123456@",
			match: false,
		},
		{
			// 巴拉巴拉
			name:  "合法邮箱",
			email: "123456@qq.com",
			match: true,
		},
	}

	h := NewUserHandler(nil, nil, nil)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			match, err := h.emailRexExp.MatchString(tc.email)
			// note testify 库中的 require 断言。若失败则终止测试，t.Fatal(err)也会中断
			require.NoError(t, err)
			assert.NoError(t, err)
			// note testify 库中的 assert 断言。若失败则不会停止测试，会标记失败并记录信息
			assert.Equal(t, tc.match, match)
		})
	}
}
