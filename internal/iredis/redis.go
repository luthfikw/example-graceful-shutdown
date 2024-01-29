package iredis

import (
	"context"

	"github.com/go-redis/redis/v8"
)

func NewRedis() (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
	})

	result := client.Ping(context.Background())
	if err := result.Err(); err != nil && err != redis.Nil {
		return nil, result.Err()
	}

	return client, nil
}
