package testinfra

import (
	"context"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/nats"
)

const NATSImage = "nats:2-alpine"

// StartNATS starts nats-server with JetStream enabled. URI = "nats://...".
func StartNATS(ctx context.Context) (*Container, error) {
	c, err := nats.Run(ctx, NATSImage) // the module passes -js by default
	if err != nil {
		return nil, err
	}
	uri, err := c.ConnectionString(ctx)
	if err != nil {
		_ = testcontainers.TerminateContainer(c)
		return nil, err
	}
	return &Container{Container: c, URI: uri}, nil
}
