package health

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// RedisCheckerConfig holds configuration for the RedisChecker.
type RedisCheckerConfig struct {
	Name string
}

// RedisChecker implements the HealthChecker interface for Redis.
type RedisChecker struct {
	config RedisCheckerConfig
	client redis.Cmdable
}

// NewRedisChecker creates a new RedisChecker using an existing redis.Cmdable client.
func NewRedisChecker(client redis.Cmdable, config RedisCheckerConfig) (*RedisChecker, error) {
	if client == nil {
		return nil, fmt.Errorf("failed to create redis client: %w", fmt.Errorf("no redis interface detected"))
	}

	return &RedisChecker{
		config: config, client: client}, nil
}

// Check performs the Redis health check.
func (c *RedisChecker) Check(ctx context.Context) error {
	_, err := c.client.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("redis is down: %w", err)
	}
	return nil
}

// Name returns the name of the health check.
func (c *RedisChecker) Name() string {
	return c.config.Name
}

// Type returns the type of the health check (redis).
func (c *RedisChecker) Type() string {
	return "redis"
}
