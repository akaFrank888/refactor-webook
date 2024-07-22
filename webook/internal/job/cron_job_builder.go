package job

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/robfig/cron/v3"
	"refactor-webook/webook/pkg/logger"
	"strconv"
	"time"
)

type CronJobBuilder struct {
	l logger.LoggerV1
	// 接入 prometheus 统计响应时间
	vector *prometheus.SummaryVec
}

func NewCronJobBuilder(l logger.LoggerV1, opt prometheus.SummaryOpts) *CronJobBuilder {
	vector := prometheus.NewSummaryVec(opt, []string{"job", "success"})
	return &CronJobBuilder{l: l, vector: vector}
}

// Build Builder模式 传入自己的 job 构建出 cron.Job
func (b *CronJobBuilder) Build(job Job) cron.Job {
	return cronJobAdapterFunc(func() {
		// 统计执行时间
		start := time.Now()

		err := job.Run()
		if err != nil {
			b.l.Error("执行失败",
				logger.Error(err),
				logger.String("name", job.Name()))
		}
		defer func() {
			duration := time.Since(start)
			b.vector.WithLabelValues(job.Name(), strconv.FormatBool(err == nil)).
				Observe(float64(duration.Milliseconds()))
		}()
	})
}

type cronJobAdapterFunc func()

func (c cronJobAdapterFunc) Run() {
	c()
}
