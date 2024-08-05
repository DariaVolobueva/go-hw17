package cache

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisCache struct {
    client *redis.Client
}

func NewRedisCache() *RedisCache {
    client := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    return &RedisCache{client: client}
}

func (rc *RedisCache) Get(ctx context.Context, key string) (string, error) {
    return rc.client.Get(ctx, key).Result()
}

func (rc *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
    return rc.client.Set(ctx, key, value, expiration).Err()
}

func (rc *RedisCache) Del(ctx context.Context, key string) error {
    return rc.client.Del(ctx, key).Err()
}