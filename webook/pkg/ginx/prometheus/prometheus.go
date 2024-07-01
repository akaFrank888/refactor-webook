package prometheus

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"strconv"
	"time"
)

type Builder struct {
	Name       string
	Namespace  string
	Subsystem  string
	InstanceID string
	Help       string
}

func NewBuilder(name string, namespace string, subsystem string, instanceID string, help string) *Builder {
	return &Builder{Name: name, Namespace: namespace, Subsystem: subsystem, InstanceID: instanceID, Help: help}
}

// BuildResponseTime 统计http请求响应时间
func (b *Builder) BuildResponseTime() gin.HandlerFunc {
	// 分 请求方法、命中的路由和响应码
	labels := []string{"method", "pattern", "status"}
	vector := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		// note 这三个都不能有除了下划线以外的字符！！！
		Namespace: b.Namespace,
		Subsystem: b.Subsystem,
		Name:      b.Name + "_resp_time",
		Help:      b.Help,
		ConstLabels: map[string]string{
			// 部署到了多个实例
			"instance_id": b.InstanceID,
		},
		Objectives: map[float64]float64{
			0.5:   0.01,
			0.75:  0.01,
			0.90:  0.01,
			0.99:  0.001,
			0.999: 0.0001,
		},
	}, labels)
	prometheus.MustRegister(vector)
	return func(ctx *gin.Context) {
		// 取当前时间
		start := time.Now()
		// note 当执行完ctx.Next()后，控制权回到此中间件BuildResponseTime处，执行defer
		defer func() {
			// 用于上报 prometheus
			duration := time.Since(start).Milliseconds()
			method := ctx.Request.Method
			pattern := ctx.FullPath()
			status := ctx.Writer.Status()
			vector.WithLabelValues(method, pattern, strconv.Itoa(status)).Observe(float64(duration))
		}()

		// 执行下一个middleware
		ctx.Next()
	}
}

// BuildActiveRequest 统计活跃请求数
func (b *Builder) BuildActiveRequest() gin.HandlerFunc {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		// note 这三个都不能有除了下划线以外的字符！！！
		Namespace: b.Namespace,
		Subsystem: b.Subsystem,
		Name:      b.Name + "_active_request",
		Help:      b.Help,
		ConstLabels: map[string]string{
			// 部署到了多个实例
			"instance_id": b.InstanceID,
		},
	})

	prometheus.MustRegister(gauge)
	return func(ctx *gin.Context) {
		gauge.Inc()
		defer gauge.Dec()

		ctx.Next()
	}
}
