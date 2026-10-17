package testinfra

import (
	"context"
	"strings"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/redis"
)

const RedisImage = "redis:8-alpine"

// StartRedis returns a container with URI = "host:port".
func StartRedis(ctx context.Context) (*Container, error) {
	c, err := redis.Run(ctx, RedisImage)
	if err != nil {
		return nil, err
	}
	uri, err := c.ConnectionString(ctx)
	if err != nil {
		_ = testcontainers.TerminateContainer(c)
		return nil, err
	}
	return &Container{Container: c, URI: strings.TrimPrefix(uri, "redis://")}, nil
}
