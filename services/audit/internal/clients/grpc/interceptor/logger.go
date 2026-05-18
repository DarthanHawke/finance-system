package interceptor

/* legacy code
import (
	"context"

	grpclog "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"

	"go.uber.org/zap"
)

func InterceptorLogger(l *zap.Logger) grpclog.Logger {
	return grpclog.LoggerFunc(func(ctx context.Context, level grpclog.Level, msg string, fields ...any) {
		zapLevel := zap.InfoLevel
		switch level {
		case grpclog.LevelDebug:
			zapLevel = zap.DebugLevel
		case grpclog.LevelInfo:
			zapLevel = zap.InfoLevel
		case grpclog.LevelWarn:
			zapLevel = zap.WarnLevel
		case grpclog.LevelError:
			zapLevel = zap.ErrorLevel
		}

		// Преобразуем fields в zap.Field
		zapFields := make([]zap.Field, 0, len(fields)/2)
		for i := 0; i < len(fields); i += 2 {
			if i+1 >= len(fields) {
				break
			}
			key, ok := fields[i].(string)
			if !ok {
				continue
			}
			zapFields = append(zapFields, zap.Any(key, fields[i+1]))
		}

		// Логируем сообщение
		if ce := l.Check(zapLevel, msg); ce != nil {
			ce.Write(zapFields...)
		}
	})
}
*/
