package interceptors

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func Logging() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		code := status.Code(err)

		level := slog.LevelInfo
		if InfraPublicMethods[info.FullMethod] {
			level = slog.LevelDebug
		}
		slog.Log(ctx, level, "rpc",
			"method", info.FullMethod,
			"duration_ms", time.Since(start).Milliseconds(),
			"code", code.String(),
		)
		return resp, err
	}
}
