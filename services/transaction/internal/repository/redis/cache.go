package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"transaction-service/internal/lib/metrics"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Cache предоставляет операции кэширования
type Cache struct {
	client *Client
	ttl    time.Duration
	tracer trace.Tracer
}

// NewCache создает новый экземпляр кэша
func NewCache(client *Client, ttl time.Duration) *Cache {
	return &Cache{
		client: client,
		ttl:    ttl,
		tracer: otel.Tracer("redis-cache"),
	}
}

// Get получает значение из кэша
func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	start := time.Now()
	defer func() {
		metrics.RedisCommandDuration.WithLabelValues("get").Observe(time.Since(start).Seconds())
	}()

	ctx, span := c.tracer.Start(ctx, "redis.Get",
		trace.WithAttributes(attribute.String("db.key", key)),
	)
	defer span.End()

	val, err := c.client.get(ctx, key)
	if err != nil {
		span.RecordError(err)
		metrics.CacheMissTotal.Inc()
		span.SetStatus(codes.Error, "cache get failed")
	} else {
		metrics.CacheHitTotal.Inc()
	}
	return val, err
}

// GetJSON получает и десериализует JSON из кэша
func (c *Cache) GetJSON(ctx context.Context, key string, dest any) error {
	start := time.Now()
	defer func() {
		metrics.RedisCommandDuration.WithLabelValues("get").Observe(time.Since(start).Seconds())
	}()

	ctx, span := c.tracer.Start(ctx, "redis.GetJSON",
		trace.WithAttributes(attribute.String("db.key", key)),
	)
	defer span.End()

	data, err := c.client.get(ctx, key)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "cache get failed")
		metrics.CacheMissTotal.Inc()
		return err
	}
	metrics.CacheHitTotal.Inc()
	return json.Unmarshal([]byte(data), dest)
}

// Set сохраняет значение в кэш с TTL по умолчанию
func (c *Cache) Set(ctx context.Context, key string, value any) error {
	start := time.Now()
	defer func() {
		metrics.RedisCommandDuration.WithLabelValues("set").Observe(time.Since(start).Seconds())
	}()

	ctx, span := c.tracer.Start(ctx, "redis.Set",
		trace.WithAttributes(attribute.String("db.key", key)),
	)
	defer span.End()

	err := c.client.set(ctx, key, value, c.ttl)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "cache set failed")
	}
	return err
}

// SetJSON сериализует в JSON и сохраняет
func (c *Cache) SetJSON(ctx context.Context, key string, value any) error {
	start := time.Now()
	defer func() {
		metrics.RedisCommandDuration.WithLabelValues("set").Observe(time.Since(start).Seconds())
	}()

	ctx, span := c.tracer.Start(ctx, "redis.SetJSON",
		trace.WithAttributes(attribute.String("db.key", key)),
	)
	defer span.End()

	data, err := json.Marshal(value)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "marshal failed")
		return fmt.Errorf("cache marshal: %w", err)
	}
	return c.client.set(ctx, key, data, c.ttl)
}

// SetWithTTL сохраняет с указанным TTL
func (c *Cache) SetWithTTL(ctx context.Context, key string, value any, ttl time.Duration) error {
	start := time.Now()
	defer func() {
		metrics.RedisCommandDuration.WithLabelValues("set").Observe(time.Since(start).Seconds())
	}()

	ctx, span := c.tracer.Start(ctx, "redis.SetWithTTL",
		trace.WithAttributes(attribute.String("db.key", key)),
	)
	defer span.End()

	err := c.client.set(ctx, key, value, ttl)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "cache set failed")
	}
	return err
}

// Exists проверяет наличие ключа
func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	start := time.Now()
	defer func() {
		metrics.RedisCommandDuration.WithLabelValues("exists").Observe(time.Since(start).Seconds())
	}()

	ctx, span := c.tracer.Start(ctx, "redis.Exists",
		trace.WithAttributes(attribute.String("db.key", key)),
	)
	defer span.End()

	exists, err := c.client.exists(ctx, key)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "cache exists failed")
	}
	return exists, err
}

// Delete удаляет ключ
func (c *Cache) Delete(ctx context.Context, key string) error {
	start := time.Now()
	defer func() {
		metrics.RedisCommandDuration.WithLabelValues("delete").Observe(time.Since(start).Seconds())
	}()

	ctx, span := c.tracer.Start(ctx, "redis.Delete",
		trace.WithAttributes(attribute.String("db.key", key)),
	)
	defer span.End()

	err := c.client.delete(ctx, key)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "cache delete failed")
	}
	return err
}

// DeleteByPrefix удаляет все ключи с префиксом
func (c *Cache) DeleteByPrefix(ctx context.Context, prefix string) error {
	start := time.Now()
	defer func() {
		metrics.RedisCommandDuration.WithLabelValues("delete_by_prefix").Observe(time.Since(start).Seconds())
	}()

	ctx, span := c.tracer.Start(ctx, "redis.DeleteByPrefix",
		trace.WithAttributes(attribute.String("db.key_prefix", prefix)),
	)
	defer span.End()

	err := c.client.deleteByPrefix(ctx, prefix)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "cache delete by prefix failed")
	}
	return err
}
