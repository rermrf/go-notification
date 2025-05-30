package circuitbreaker

import (
	"context"
	"github.com/go-kratos/aegis/circuitbreaker"
	"go-notification/internal/errs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Builder struct {
	breaker circuitbreaker.CircuitBreaker
}

func (b *Builder) Build() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		err = b.breaker.Allow()
		if err != nil {
			b.breaker.MarkFailed()
			return nil, status.Errorf(codes.Unavailable, "%s", errs.ErrCircuitBreaker)
		}

		resp, err = handler(ctx, req)

		if err != nil {
			// 对错误断言，找到代表服务端出现故障的错误，才 MarkFailed
			b.breaker.MarkFailed()
			return
		}

		b.breaker.MarkSuccess()
		return
	}
}
