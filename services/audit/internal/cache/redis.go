package cache

/* legacy code
import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
	ttl    time.Duration // Время жизни кеша
}

func NewRedisCache(addr string, password string, db int, ttl time.Duration) *RedisCache {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &RedisCache{
		client: client,
		ttl:    ttl,
	}
}

// Get Получаем значения из кеша
func (r *RedisCache) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

// Exists проверяет существует ли значение в кэше
func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	exists, err := r.client.Exists(ctx, key).Result()
	return exists == 1, err
}

// Set Сохраняем в кеш
func (r *RedisCache) Set(ctx context.Context, key string, value any) error {
	return r.client.Set(ctx, key, value, r.ttl).Err()
}

// SetWithTTL Сохраняем в кеш с заданным ttl
func (r *RedisCache) SetWithTTL(ctx context.Context, key string, value any, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

// Delete Удаляем из кэша
func (r *RedisCache) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

// DeleteByPrefix Удаляет все ключи, начинающиеся с указанного префикса
func (r *RedisCache) DeleteByPrefix(ctx context.Context, prefix string) error {
	var cursor uint64
	var keys []string
	var err error

	// Используем SCAN для итерации по всем ключам с заданным префиксом
	for {
		keys, cursor, err = r.client.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			return err
		}

		// Если нашли ключи - удаляем их
		if len(keys) > 0 {
			if err := r.client.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}

		// Завершаем итерацию когда cursor вернет 0
		if cursor == 0 {
			break
		}
	}

	return nil
}

func (r *RedisCache) Close() error {
	return r.client.Close()
}
*/
