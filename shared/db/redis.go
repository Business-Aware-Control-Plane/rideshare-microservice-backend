package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"ride-sharing/shared/retry"
)

// NewRedisClient creates a new Redis client with retry logic
func NewRedisClient(addr, password string) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Try pinging with backoff
	err := retry.WithBackoff(ctx, retry.DefaultConfig(), func() error {
		return rdb.Ping(ctx).Err()
	})
	if err != nil {
		rdb.Close()
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Println("Successfully connected to Redis")
	return rdb, nil
}
