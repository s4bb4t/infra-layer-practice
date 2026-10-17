// Package watermillx adapts Watermill publishers/subscribers to our ports.
//
// Watermill hides broker differences behind message.Publisher and
// message.Subscriber; this package hides Watermill behind port.EventPublisher
// and port.EventSubscriber. Any Watermill backend (AMQP, Redis Streams, SQL,
// Kafka, ...) becomes an adapter in a few lines: see packages rabbit and redisstreams.
package watermillx

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/s4bb4t/infra-layer-practice/internal/app/domain"
	"github.com/s4bb4t/infra-layer-practice/internal/app/port"
)

// ToMessage puts the whole envelope into the payload. Watermill's UUID is the
// event ID, so deduplication keys are the same on every backend.
func ToMessage(e domain.Event) (*message.Message, error) {
	raw, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}
	msg := message.NewMessage(e.ID, raw)
	msg.Metadata.Set("type", string(e.Type))
	msg.Metadata.Set("aggregate_id", string(e.AggregateID))
	return msg, nil
}

func FromMessage(msg *message.Message) (domain.Event, error) {
	var e domain.Event
	if err := json.Unmarshal(msg.Payload, &e); err != nil {
		return domain.Event{}, fmt.Errorf("decode event %s: %w", msg.UUID, err)
	}
	return e, nil
}

// Publisher implements port.EventPublisher on any Watermill publisher.
type Publisher struct {
	Pub   message.Publisher
	Topic string
}

var _ port.EventPublisher = Publisher{}

func (p Publisher) Publish(ctx context.Context, events ...domain.Event) error {
	msgs := make([]*message.Message, len(events))
	for i, e := range events {
		m, err := ToMessage(e)
		if err != nil {
			return err
		}
		m.SetContext(ctx)
		msgs[i] = m
	}
	return p.Pub.Publish(p.Topic, msgs...)
}

// Consume runs h for every message until msgs is closed. Ack on success, Nack
// on failure: the backend redelivers. A message that cannot be decoded is
// acked and dropped: redelivering it would fail forever (a "poison message").
func Consume(ctx context.Context, msgs <-chan *message.Message, h port.EventHandler) {
	for msg := range msgs {
		e, err := FromMessage(msg)
		if err != nil {
			msg.Ack()
			continue
		}
		if err := h(ctx, e); err != nil {
			msg.Nack()
			continue
		}
		msg.Ack()
	}
}
