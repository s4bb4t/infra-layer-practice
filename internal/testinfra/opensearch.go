package testinfra

import (
	"context"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/opensearch"
)

const OpenSearchImage = "opensearchproject/opensearch:2.19.2"

// StartOpenSearch starts a single node with the security plugin off. URI = "http://host:port".
func StartOpenSearch(ctx context.Context) (*Container, error) {
	c, err := opensearch.Run(ctx, OpenSearchImage)
	if err != nil {
		return nil, err
	}
	addr, err := c.Address(ctx)
	if err != nil {
		_ = testcontainers.TerminateContainer(c)
		return nil, err
	}
	return &Container{Container: c, URI: addr}, nil
}
