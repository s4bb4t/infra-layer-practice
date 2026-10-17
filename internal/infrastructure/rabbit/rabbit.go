// Package rabbit implements the broker ports on RabbitMQ through Watermill.
//
// Topology (Watermill "durable pub/sub"): one fanout exchange per topic, one
// durable queue per consumer group bound to it. Fan-out between groups comes
// from the exchange, competing consumers from several consumers on one queue.
package rabbit

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"

	"github.com/s4bb4t/infra-layer-practice/internal/app/port"
	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/watermillx"
)

var (
	_ port.EventPublisher  = (*Broker)(nil)
	_ port.EventSubscriber = (*Broker)(nil)
)

type Broker struct {
	uri    string
	topic  string
	logger watermill.LoggerAdapter
	pub    *amqp.Publisher
	watermillx.Publisher

	mu   sync.Mutex
	subs []io.Closer
}

func New(uri, topic string, logger watermill.LoggerAdapter) (*Broker, error) {
	if logger == nil {
		logger = watermill.NopLogger{}
	}
	// The queue name generator is irrelevant for the publisher: it only
	// declares the exchange.
	pub, err := amqp.NewPublisher(amqp.NewDurablePubSubConfig(uri, nil), logger)
	if err != nil {
		return nil, fmt.Errorf("amqp publisher: %w", err)
	}
	return &Broker{
		uri: uri, topic: topic, logger: logger, pub: pub,
		Publisher: watermillx.Publisher{Pub: pub, Topic: topic},
	}, nil
}

// Subscribe declares <topic>_<group> queue and binds it before returning.
func (b *Broker) Subscribe(ctx context.Context, group string, h port.EventHandler) error {
	cfg := amqp.NewDurablePubSubConfig(b.uri, amqp.GenerateQueueNameTopicNameWithSuffix(group))
	sub, err := amqp.NewSubscriber(cfg, b.logger)
	if err != nil {
		return fmt.Errorf("amqp subscriber: %w", err)
	}
	msgs, err := sub.Subscribe(ctx, b.topic)
	if err != nil {
		_ = sub.Close()
		return fmt.Errorf("subscribe: %w", err)
	}
	b.track(sub)
	go func() {
		watermillx.Consume(ctx, msgs, h)
		_ = sub.Close()
	}()
	return nil
}

func (b *Broker) track(sub io.Closer) {
	b.mu.Lock()
	b.subs = append(b.subs, sub)
	b.mu.Unlock()
}

// Close stops every subscription of this broker, then the publisher.
// Without it, subscribers keep reconnecting after the broker is gone.
func (b *Broker) Close() error {
	b.mu.Lock()
	subs := b.subs
	b.subs = nil
	b.mu.Unlock()
	for _, s := range subs {
		_ = s.Close()
	}
	return b.pub.Close()
}
