package testinfra

import (
	"context"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/rabbitmq"
)

const RabbitImage = "rabbitmq:4-management-alpine"

// StartRabbit returns a container with URI = "amqp://...".
func StartRabbit(ctx context.Context) (*Container, error) {
	c, err := rabbitmq.Run(ctx, RabbitImage)
	if err != nil {
		return nil, err
	}
	uri, err := c.AmqpURL(ctx)
	if err != nil {
		_ = testcontainers.TerminateContainer(c)
		return nil, err
	}
	return &Container{Container: c, URI: uri}, nil
}
