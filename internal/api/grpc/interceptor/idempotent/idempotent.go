package idempotent

import (
	"context"
	"fmt"
	notificationv1 "go-notification/api/proto/gen/api/proto/notification/v1"
	"go-notification/internal/api/grpc/interceptor/jwt"
	"go-notification/internal/pkg/idempotent"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Builder struct {
	svc idempotent.IdempotencyService
}

func NewBuilder(svc idempotent.IdempotencyService) *Builder {
	return &Builder{svc: svc}
}

func (b *Builder) Build() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		bizID, err := jwt.GetBizIDFromContext(ctx)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "%v", err)
		}
		v, ok := req.(notificationv1.IdempotencyCarrier)
		if !ok {
			return handler(ctx, req)
		}
		// 需要幂等检测
		keys := v.GetIdempotencyKeys()
		var key1s []string
		for _, key := range keys {
			key1s = append(key1s, fmt.Sprintf("%d-%s", bizID, key))
		}
		exists, err := b.svc.MExists(ctx, key1s...)
		if err != nil {
			return nil, fmt.Errorf("进行幂等检测失败")
		}
		for idx := range exists {
			if exists[idx] {
				return nil, status.Errorf(codes.InvalidArgument, "%v", fmt.Errorf("幂等检测没通过，有重复请求"))
			}
		}
		return handler(ctx, req)
	}
}
