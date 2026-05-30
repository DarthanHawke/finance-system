package tracing

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

// WithTraceContext возвращает логгер с trace_id и span_id из контекста.
func WithTraceContext(ctx context.Context, logger *slog.Logger) *slog.Logger {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().HasTraceID() {
		return logger
	}

	attrs := []any{
		"trace_id", span.SpanContext().TraceID().String(),
		"span_id", span.SpanContext().SpanID().String(),
	}

	return logger.With(attrs...)
}
