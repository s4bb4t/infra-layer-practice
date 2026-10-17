package testinfra

import (
	"context"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/kafka"
)

const KafkaImage = "confluentinc/confluent-local:7.5.0" // the version the testcontainers module is tested with

// StartKafka starts a single KRaft broker. URI = "host:port" bootstrap address.
func StartKafka(ctx context.Context) (*Container, error) {
	c, err := kafka.Run(ctx, KafkaImage, kafka.WithClusterID("orders-test"))
	if err != nil {
		return nil, err
	}
	brokers, err := c.Brokers(ctx)
	if err != nil {
		_ = testcontainers.TerminateContainer(c)
		return nil, err
	}
	return &Container{Container: c, URI: brokers[0]}, nil
}
