package testinfra

import (
	"context"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

const PostgresImage = "postgres:17-alpine"

// StartPostgres returns a container with URI = "postgres://..." DSN.
func StartPostgres(ctx context.Context) (*Container, error) {
	c, err := postgres.Run(ctx, PostgresImage,
		postgres.WithDatabase("orders"),
		postgres.WithUsername("orders"),
		postgres.WithPassword("orders"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		return nil, err
	}
	dsn, err := c.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = testcontainers.TerminateContainer(c)
		return nil, err
	}
	return &Container{Container: c, URI: dsn}, nil
}
