package testinfra

import (
	"context"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
)

const MySQLImage = "mysql:8.4"

// StartMySQL returns a container with URI = go-sql-driver DSN
// (parseTime and UTC are already set).
func StartMySQL(ctx context.Context) (*Container, error) {
	c, err := mysql.Run(ctx, MySQLImage,
		mysql.WithDatabase("orders"),
		mysql.WithUsername("orders"),
		mysql.WithPassword("orders"),
	)
	if err != nil {
		return nil, err
	}
	dsn, err := c.ConnectionString(ctx, "parseTime=true", "loc=UTC", "multiStatements=true")
	if err != nil {
		_ = testcontainers.TerminateContainer(c)
		return nil, err
	}
	return &Container{Container: c, URI: dsn}, nil
}
