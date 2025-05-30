package timeout

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const timeoutKey = "timeout"

func InjectorInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// 1. 从context提取超时时间
		if deadline, ok := ctx.Deadline(); ok {
			// 2. 注入metadata（兼容已有metadata）
			ctx = metadata.AppendToOutgoingContext(ctx, timeoutKey, fmt.Sprintf("%d", deadline.UnixMilli()))
		}
		// 3. 透传修改后的 context
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
