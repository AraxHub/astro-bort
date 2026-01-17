package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/admin/tg-bots/astro-bot/internal/ports/cache"
	"github.com/redis/go-redis/v9"
)

// Client обёртка над redis.Client для работы с кэшем
// Реализует интерфейс cache.Cache
type Client struct {
	client *redis.Client
}

// NewClient создаёт новый Redis-клиент
func NewClient(client *redis.Client) cache.Cache {
	return &Client{
		client: client,
	}
}

// Get получает значение по ключу
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("key not found: %s", key)
	}
	if err != nil {
		return "", fmt.Errorf("redis get failed: %w", err)
	}
	return val, nil
}

// Set устанавливает значение с TTL
func (c *Client) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	if err := c.client.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("redis set failed: %w", err)
	}
	return nil
}

// Delete удаляет значение по ключу
func (c *Client) Delete(ctx context.Context, key string) error {
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis delete failed: %w", err)
	}
	return nil
}

// Exists проверяет существование ключа
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	count, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists failed: %w", err)
	}
	return count > 0, nil
}

// Close закрывает подключение к кэшу
func (c *Client) Close() error {
	return c.client.Close()
}
