package redis

import (
	"context"
	"fmt"
	"time"
	"transaction-service/internal/lib/metrics"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Deduplicator обеспечивает защиту от дубликатов
type Deduplicator struct {
	client *Client
	ttl    time.Duration
	tracer trace.Tracer
}

// NewDeduplicator создает новый дедупликатор
func NewDeduplicator(client *Client, ttl time.Duration) *Deduplicator {
	return &Deduplicator{
		client: client,
		ttl:    ttl,
		tracer: otel.Tracer("redis-deduplicator"),
	}
}

// IsDuplicate проверяет, был ли ключ уже обработан
func (d *Deduplicator) IsDuplicate(ctx context.Context, key string) (bool, error) {
	start := time.Now()
	defer func() {
		metrics.RedisCommandDuration.WithLabelValues("setnx").Observe(time.Since(start).Seconds())
	}()

	ctx, span := d.tracer.Start(ctx, "redis.IsDuplicate",
		trace.WithAttributes(attribute.String("db.key", key)),
	)
	defer span.End()

	ok, err := d.client.setNX(ctx, key, "processed", d.ttl)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "dedup check failed")
		return false, fmt.Errorf("dedup check: %w", err)
	}
	return !ok, nil
}

// MarkProcessed помечает ключ как обработанный
func (d *Deduplicator) MarkProcessed(ctx context.Context, key string) error {
	start := time.Now()
	defer func() {
		metrics.RedisCommandDuration.WithLabelValues("set").Observe(time.Since(start).Seconds())
	}()

	ctx, span := d.tracer.Start(ctx, "redis.MarkProcessed",
		trace.WithAttributes(attribute.String("db.key", key)),
	)
	defer span.End()

	err := d.client.set(ctx, key, "processed", d.ttl)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "mark processed failed")
	}
	return err
}

// IsProcessed проверяет, обработан ли ключ (без установки)
func (d *Deduplicator) IsProcessed(ctx context.Context, key string) (bool, error) {
	start := time.Now()
	defer func() {
		metrics.RedisCommandDuration.WithLabelValues("exists").Observe(time.Since(start).Seconds())
	}()

	ctx, span := d.tracer.Start(ctx, "redis.IsProcessed",
		trace.WithAttributes(attribute.String("db.key", key)),
	)
	defer span.End()

	exists, err := d.client.exists(ctx, key)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "check processed failed")
	}
	return exists, err
}

// Remove удаляет ключ (для возможности повторной обработки)
func (d *Deduplicator) Remove(ctx context.Context, key string) error {
	start := time.Now()
	defer func() {
		metrics.RedisCommandDuration.WithLabelValues("delete").Observe(time.Since(start).Seconds())
	}()

	ctx, span := d.tracer.Start(ctx, "redis.Remove",
		trace.WithAttributes(attribute.String("db.key", key)),
	)
	defer span.End()

	err := d.client.delete(ctx, key)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "dedup remove failed")
	}
	return err
}
