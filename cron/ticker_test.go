package cron

import (
	"context"
	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
	"time"
)

func TestTicker(t *testing.T) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	for {
		select {
		case <-ctx.Done(): // 超时的情况
			// select 中不要使用 break
			return

		case now := <-ticker.C: // note ticker.C是只读的 channel，每 1s 会产生一个 Time 类型的数据
			log.Println("现在是：", now.UnixMilli())
		}
	}
}

func TestCronExpr(t *testing.T) {
	expr := cron.New(cron.WithSeconds())

	// 填写 cron 表达式
	id, err := expr.AddFunc("@every 1s", func() {
		t.Log("执行了")
	})
	assert.NoError(t, err)
	t.Log("任务", id)

	// 开始调度任务
	expr.Start()
	time.Sleep(time.Second * 10)

	ctx := expr.Stop() // 意思是，你不要调度新任务执行了，你正在执行的继续执行
	t.Log("发出来停止信号")

	<-ctx.Done()
	t.Log("彻底停下来了，没有任务在执行")
	// 这边，彻底停下来了
}
