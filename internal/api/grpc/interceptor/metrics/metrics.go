package metrics

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
)

const (
	// 分位数常量
	percentile50 float64 = 0.5
	percentile90 float64 = 0.9
	percentile99 float64 = 0.99

	// 误差边界常量
	errorMargin50 float64 = 0.05
	errorMargin90 float64 = 0.01
	errorMargin99 float64 = 0.001
)

type Builder struct {
	// apiDurationSummary 跟踪 API 响应时间
	apiDurationSummary *prometheus.SummaryVec
	// requestCounter 跟踪请求总数
	requestCounter *prometheus.CounterVec
	// errorCounter 跟踪失败请求数
	errorCounter *prometheus.CounterVec
}

// NewBuilder 创建一个带有初始化指标的 Builder
func NewBuilder() *Builder {
	return &Builder{
		apiDurationSummary: promauto.NewSummaryVec(prometheus.SummaryOpts{
			Name: "grpc_server_handling_seconds",
			Help: "gRPC请求的响应延迟（秒）摘要。",
			Objectives: map[float64]float64{
				percentile50: errorMargin50,
				percentile90: errorMargin90,
				percentile99: errorMargin99,
			},
		}, []string{"method", "status_code"}),
		requestCounter: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "grpc_server_request_total",
				Help: "收到的gRPC请求总数。",
			}, []string{"method"},
		),
		errorCounter: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "grpc_server_errors_total",
				Help: "导致错误的gRPC请求总数。",
			}, []string{"method", "status"}),
	}
}

func (b *Builder) Build() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 记录开始时间
		startTime := time.Now()

		// 增加请求计数器
		b.requestCounter.WithLabelValues(info.FullMethod).Inc()

		// 处理请求
		resp, err := handler(ctx, req)

		// 计算持续时间
		duration := time.Since(startTime).Seconds()

		// 获取状态码
		st, _ := status.FromError(err)
		statusCode := status.Code(err).String()

		// 如果出现错误，则增加错误计数器
		if st.Code() != codes.OK {
			b.requestCounter.WithLabelValues(info.FullMethod, statusCode).Inc()
		}

		// 向 Prometheus 报告
		b.apiDurationSummary.WithLabelValues(info.FullMethod, statusCode).Observe(duration)
		return resp, err
	}
}
