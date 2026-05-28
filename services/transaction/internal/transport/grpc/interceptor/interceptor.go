package grpc

import (
	"context"
	"log/slog"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnaryServerInterceptor возвращает интерцептор для логирования и восстановления после паники
func UnaryServerInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		// Обработка паники
		defer func() {
			if p := recover(); p != nil {
				logger.ErrorContext(ctx, "panic in gRPC handler",
					"method", info.FullMethod,
					"panic", p,
					"stack", string(debug.Stack()),
				)
				err = status.Error(codes.Internal, "internal error")
			}
		}()

		resp, err = handler(ctx, req)
		if err != nil {
			st, _ := status.FromError(err)
			if st.Code() == codes.Internal {
				logger.ErrorContext(ctx, "gRPC request failed",
					"method", info.FullMethod,
					"code", st.Code().String(),
				)
			} else {
				logger.WarnContext(ctx, "gRPC request failed",
					"method", info.FullMethod,
					"code", st.Code().String(),
				)
			}
		}

		return resp, err
	}
}
