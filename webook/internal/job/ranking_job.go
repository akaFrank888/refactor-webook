package job

import (
	"context"
	"refactor-webook/webook/internal/service"
	"time"
)

type RankingJob struct {
	svc     service.RankingService
	timeout time.Duration
}

func NewRankingJob(svc service.RankingService, timeout time.Duration) *RankingJob {
	return &RankingJob{svc: svc, timeout: timeout}
}

func (r *RankingJob) Name() string {
	return "RankingJob"
}

func (r *RankingJob) Run() error {
	// 每一次 job 的过期时间是 30s
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()
	return r.svc.TopN(ctx)
}
