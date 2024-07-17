package prometheus

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"refactor-webook/webook/internal/sms_project/service"
	"time"
)

type Decorator struct {
	svc    service.Service
	vector *prometheus.SummaryVec
}

func NewDecorator(svc service.Service, opt prometheus.SummaryOpts) *Decorator {
	return &Decorator{
		svc:    svc,
		vector: prometheus.NewSummaryVec(opt, []string{"tpl_id"}),
	}
}

func (d *Decorator) Send(ctx context.Context,
	tplId string, args []string, numbers ...string) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		d.vector.WithLabelValues(tplId).Observe(float64(duration))
	}()
	return d.svc.Send(ctx, tplId, args, numbers...)
}
