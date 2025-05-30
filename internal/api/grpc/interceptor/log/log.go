package log

import (
	"context"
	"encoding/json"
	"go-notification/internal/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"time"
)

// Builder 日志拦截器构建器
type Builder struct {
	logger logger.Logger
}

func NewBuilder() *Builder {
	zapLogger, _ := zap.NewDevelopment()
	return &Builder{
		logger: logger.NewZapLogger(zapLogger),
	}
}

func (b *Builder) WithLogger(logger logger.Logger) *Builder {
	b.logger = logger
	return b
}

func (b *Builder) Build() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 记录开始时间
		startTime := time.Now()

		// 将请求对象转为 JSON 字符串进行记录
		reqJSON, _ := json.Marshal(req)
		b.logger.Info("gRPC request",
			logger.String("method", info.FullMethod),
			logger.String("request", string(reqJSON)),
			logger.Any("start_time", startTime))

		// 处理请求
		resp, err := handler(ctx, req)

		// 计算请求处理时间
		duration := time.Since(startTime)

		// 获取状态码
		st, _ := status.FromError(err)
		statusCode := st.Code()

		// 将响应对象转为 JSON 字符串进行记录
		respJSON, _ := json.Marshal(resp)

		if err != nil {
			// 如果有错误，记录错误日志
			b.logger.Error("gRPC response with error",
				logger.String("method", info.FullMethod),
				logger.String("status_code", statusCode.String()),
				logger.String("response", string(respJSON)),
				logger.Any("duration", duration),
				logger.Error(err))
		} else {
			// 记录成功响应
			b.logger.Info("gRPC response",
				logger.String("method", info.FullMethod),
				logger.String("status_code", statusCode.String()),
				logger.String("response", string(respJSON)),
				logger.Any("duration", duration))
		}
		return resp, err
	}
}
