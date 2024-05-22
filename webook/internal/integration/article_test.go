package integration

import "testing"

func TestArticleHandle_Edit(t *testing.T) {
	testcases := []struct {
		name string
		// 提前准备数据
		before func(t *testing.T)
		// 验证和删除数据
		after func(t *testing.T)

		// todo 没懂
		art Article

		wantCode   int
		wantResult Result[int64]
	}{}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {

		})
	}

}

type Article struct {
}
type Result[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}
