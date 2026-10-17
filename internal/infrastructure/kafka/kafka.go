// Package kafka implements the broker ports on Apache Kafka with franz-go.
//
// Mapping: topic = topic with Partitions partitions, group = Kafka consumer
// group. Records are keyed by aggregate_id: all events of one order land in
// one partition and are consumed in order.
//
// Kafka has no per-message ack/nack: a consumer commits an OFFSET, "everything
// before this point is done". So a failed event cannot simply be "nacked" while
// later ones go on; this adapter retries it in place with backoff.
package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/s4bb4t/infra-layer-practice/internal/app/domain"
	"github.com/s4bb4t/infra-layer-practice/internal/app/port"
)

var (
	_ port.EventPublisher  = (*Broker)(nil)
	_ port.EventSubscriber = (*Broker)(nil)
)

const Partitions = 3

type Broker struct {
	brokers []string
	topic   string
	prod    *kgo.Client
}

// New creates the topic if needed and a producer for it.
func New(ctx context.Context, brokers []string, topic string) (*Broker, error) {
	prod, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.DefaultProduceTopic(topic),
		kgo.RequiredAcks(kgo.AllISRAcks()), // durability over latency
	)
	if err != nil {
		return nil, fmt.Errorf("kafka producer: %w", err)
	}

	adm := kadm.NewClient(prod)
	resp, err := adm.CreateTopic(ctx, Partitions, 1, nil, topic)
	if err == nil {
		err = resp.Err
	}
	if err != nil && !errors.Is(err, kerr.TopicAlreadyExists) {
		prod.Close()
		return nil, fmt.Errorf("create topic: %w", err)
	}
	return &Broker{brokers: brokers, topic: topic, prod: prod}, nil
}

func (b *Broker) Publish(ctx context.Context, events ...domain.Event) error {
	records := make([]*kgo.Record, len(events))
	for i, e := range events {
		val, err := json.Marshal(e)
		if err != nil {
			return err
		}
		records[i] = &kgo.Record{
			Key:     []byte(e.AggregateID),
			Value:   val,
			Headers: []kgo.RecordHeader{{Key: "type", Value: []byte(e.Type)}},
		}
	}
	return b.prod.ProduceSync(ctx, records...).FirstErr()
}

// Subscribe joins the group in the background. Readiness is guaranteed by
// ConsumeResetOffset(AtStart): a group without committed offsets reads the
// topic from the beginning, so nothing published after this call is missed
// even if the join takes a few seconds.
func (b *Broker) Subscribe(ctx context.Context, group string, h port.EventHandler) error {
	cl, err := kgo.NewClient(
		kgo.SeedBrokers(b.brokers...),
		kgo.ConsumerGroup(group),
		kgo.ConsumeTopics(b.topic),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
		kgo.DisableAutoCommit(), // commit only what was handled
		kgo.BlockRebalanceOnPoll(),
	)
	if err != nil {
		return fmt.Errorf("kafka consumer: %w", err)
	}
	go b.consume(ctx, cl, h)
	return nil
}

func (b *Broker) consume(ctx context.Context, cl *kgo.Client, h port.EventHandler) {
	defer cl.Close()
	for {
		fetches := cl.PollFetches(ctx)
		if ctx.Err() != nil || fetches.IsClientClosed() {
			return
		}
		fetches.EachRecord(func(r *kgo.Record) {
			var e domain.Event
			if err := json.Unmarshal(r.Value, &e); err != nil {
				return // poison record: skip it, it will never decode
			}
			for attempt := 0; ctx.Err() == nil; attempt++ {
				if err := h(ctx, e); err == nil {
					return
				}
				time.Sleep(backoff(attempt))
			}
		})
		if ctx.Err() == nil {
			_ = cl.CommitUncommittedOffsets(ctx)
		}
		cl.AllowRebalance()
	}
}

func backoff(attempt int) time.Duration {
	d := 50 * time.Millisecond << min(attempt, 6)
	return min(d, 3*time.Second)
}

func (b *Broker) Close() error {
	b.prod.Close()
	return nil
}
