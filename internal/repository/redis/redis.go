package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type Client struct {
	client *redis.Client
}

func New(ctx context.Context, addr, password string, db int) (*Client, error) {
	redisDB := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	_, err := redisDB.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("error connecting to Redis: %w", err)
	}

	log.Info().Msg("connected to Redis")

	return &Client{client: redisDB}, nil
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return "", fmt.Errorf("redis get failed: %w", err)
	}

	return val, nil
}

func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if err := c.client.Set(ctx, key, value, expiration).Err(); err != nil {
		return fmt.Errorf("error in Set(): %w", err)
	}

	return nil
}

func (c *Client) Del(ctx context.Context, keys ...string) error {
	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("error deleting good: %w", err)
	}

	return nil
}

func (c *Client) Close() error {
	if err := c.client.Close(); err != nil {
		return fmt.Errorf("error closing redis client: %w", err)
	}

	return nil
}

func (c *Client) TTL(ctx context.Context, cacheKey string) (time.Duration, error) {
	ttlCmd := c.client.TTL(ctx, cacheKey)

	ttl, err := ttlCmd.Result()
	if err != nil {
		return 0, fmt.Errorf("redis TTL failed: %w", err)
	}

	if ttl == -2 {
		return 0, redis.Nil
	}

	return ttl, nil
}
