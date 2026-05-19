package redis

import (
	"context"
	"fmt"
	"time"
)

// Deduplicator обеспечивает защиту от дубликатов
type Deduplicator struct {
	client *Client
	ttl    time.Duration
}

// NewDeduplicator создает новый дедупликатор
func NewDeduplicator(client *Client, ttl time.Duration) *Deduplicator {
	return &Deduplicator{
		client: client,
		ttl:    ttl,
	}
}

// IsDuplicate проверяет, был ли ключ уже обработан
func (d *Deduplicator) IsDuplicate(ctx context.Context, key string) (bool, error) {
	ok, err := d.client.setNX(ctx, key, "processed", d.ttl)
	if err != nil {
		return false, fmt.Errorf("dedup check: %w", err)
	}
	return !ok, nil
}

// MarkProcessed помечает ключ как обработанный
func (d *Deduplicator) MarkProcessed(ctx context.Context, key string) error {
	return d.client.set(ctx, key, "processed", d.ttl)
}

// IsProcessed проверяет, обработан ли ключ (без установки)
func (d *Deduplicator) IsProcessed(ctx context.Context, key string) (bool, error) {
	return d.client.exists(ctx, key)
}

// Remove удаляет ключ (для возможности повторной обработки)
func (d *Deduplicator) Remove(ctx context.Context, key string) error {
	return d.client.delete(ctx, key)
}
