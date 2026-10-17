package testinfra

import (
	"context"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/clickhouse"
)

const ClickHouseImage = "clickhouse/clickhouse-server:25.8-alpine"

const (
	ClickHouseUser     = "orders"
	ClickHousePassword = "orders"
)

// StartClickHouse returns a container with URI = "host:port" of the native protocol.
func StartClickHouse(ctx context.Context) (*Container, error) {
	c, err := clickhouse.Run(ctx, ClickHouseImage,
		clickhouse.WithUsername(ClickHouseUser),
		clickhouse.WithPassword(ClickHousePassword),
	)
	if err != nil {
		return nil, err
	}
	addr, err := c.ConnectionHost(ctx)
	if err != nil {
		_ = testcontainers.TerminateContainer(c)
		return nil, err
	}
	return &Container{Container: c, URI: addr}, nil
}
