// Package redisstreams implements the broker ports on Redis Streams through Watermill.
//
// Mapping: topic = stream, group = Redis consumer group (XGROUP CREATE),
// each Subscribe = one consumer in that group. Nacked or crashed messages
// stay in the group's Pending Entries List until redelivered or claimed.
package redisstreams

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	goredis "github.com/redis/go-redis/v9"

	"github.com/s4bb4t/infra-layer-practice/internal/app/port"
	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/watermillx"
	"github.com/s4bb4t/infra-layer-practice/internal/todo"
)

var (
	_ port.EventPublisher  = (*Broker)(nil)
	_ port.EventSubscriber = (*Broker)(nil)
)

type Broker struct {
	rdb    goredis.UniversalClient
	topic  string
	logger watermill.LoggerAdapter
	pub    *redisstream.Publisher
	watermillx.Publisher

	mu   sync.Mutex
	subs []io.Closer
}

func New(rdb goredis.UniversalClient, topic string, logger watermill.LoggerAdapter) (*Broker, error) {
	if logger == nil {
		logger = watermill.NopLogger{}
	}
	pub, err := redisstream.NewPublisher(redisstream.PublisherConfig{Client: rdb}, logger)
	if err != nil {
		return nil, fmt.Errorf("redisstream publisher: %w", err)
	}
	return &Broker{
		rdb: rdb, topic: topic, logger: logger, pub: pub,
		Publisher: watermillx.Publisher{Pub: pub, Topic: topic},
	}, nil
}

// Subscribe — шаг 07.
//
// TODO(step-07): посмотри rabbit.Broker.Subscribe и сделай то же на redisstream:
//   - redisstream.NewSubscriber с ConsumerGroup = group;
//   - выбери OldestId: "0" или "$"? Что получит НОВАЯ группа в каждом случае
//     и что из этого совпадает с поведением RabbitMQ-очереди? (ответ — в ADR);
//   - sub.Subscribe(ctx, b.topic) и watermillx.Consume в горутине;
//   - зарегистрируй subscriber через b.track(sub), чтобы Close() его остановил,
//     и закрой его сам, когда канал сообщений закроется.
func (b *Broker) Subscribe(ctx context.Context, group string, h port.EventHandler) error {
	return todo.Step("07")
}

//nolint:unused // used by Subscribe once step 07 is done
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
