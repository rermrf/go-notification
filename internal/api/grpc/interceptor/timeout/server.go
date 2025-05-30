package timeout

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"strconv"
	"time"
)

func TimeoutInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 1. 从metadata提取超时参数
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return handler(ctx, req) // 无 metadata 则透穿
		}

		// 2. 查找 timeout 字段
		timeoutValues := md.Get(timeoutKey)
		if len(timeoutValues) == 0 {
			return handler(ctx, req) // 无超时配置则透传
		}

		// 3. 解析超时时间
		timeoutStr := timeoutValues[0]
		timeoutStamp, err := strconv.ParseInt(timeoutStr, 10, 64)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "解析时间戳失败")
		}

		// 4. 重建带超时的 context
		newCtx, cancel := context.WithDeadline(ctx, time.UnixMilli(timeoutStamp))
		defer cancel()

		// 传递新的 context 给后续处理
		return handler(newCtx, req)
	}
}
