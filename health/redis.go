package health

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisCheckerConfig holds configuration for the RedisChecker.
type RedisCheckerConfig struct {
	Timeout time.Duration
	Name    string
}

// RedisChecker implements the HealthChecker interface for Redis.
type RedisChecker struct {
	config RedisCheckerConfig
	client redis.Cmdable
}

// NewRedisChecker creates a new RedisChecker using an existing redis.Cmdable client.
func NewRedisChecker(client redis.Cmdable, config RedisCheckerConfig) (*RedisChecker, error) {
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Second // Default timeout
	}

	// Test the connection to ensure the client is valid
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisChecker{
		config: config, client: client}, nil
}

// Check performs the Redis health check.
func (c *RedisChecker) Check(ctx context.Context) error {
	fmt.Println("performing the actual Redis check")

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

// GetTimeOut returns the timeout for the health check.
func (c *RedisChecker) GetTimeOut() time.Duration {
	return c.config.Timeout
}
