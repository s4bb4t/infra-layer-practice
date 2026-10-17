// Package kafka implements the broker ports on Apache Kafka with franz-go.
//
// Mapping: topic = topic with Partitions partitions, group = Kafka consumer
// group. Records are keyed by aggregate_id: all events of one order land in
// one partition and are consumed in order.
//
// Kafka has no per-message ack/nack: a consumer commits an OFFSET, "everything
// before this point is done". So a failed event cannot simply be "nacked" while
// later ones go on.
//
// Шаг 08: New (продюсер + создание топика) и контракт-тест даны; Publish и
// Subscribe — твои.
package kafka

import (
	"context"
	"errors"
	"fmt"

	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/s4bb4t/infra-layer-practice/internal/app/domain"
	"github.com/s4bb4t/infra-layer-practice/internal/app/port"
	"github.com/s4bb4t/infra-layer-practice/internal/todo"
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

// Publish — TODO(step-08).
//   - JSON-конверт события в Value;
//   - что положить в Key и почему? (подсказка: порядок событий одного заказа);
//   - ProduceSync или Produce с callback — в чём разница для outbox relay?
func (b *Broker) Publish(ctx context.Context, events ...domain.Event) error {
	return todo.Step("08")
}

// Subscribe — TODO(step-08).
//
// Контракт: «событие, опубликованное ПОСЛЕ возврата Subscribe, дойдёт до группы».
// Вход в consumer group в Kafka асинхронный и может занять секунды. Как выполнить
// контракт, не дожидаясь rebalance? (подсказка: kgo.ConsumeResetOffset)
//
// В Kafka нет nack на сообщение — только commit позиции. Что делать, если
// обработчик вернул ошибку? Варианты и их цена — в ADR.
//
// Не забудь: автокоммит offset'ов коммитит то, что ты ПОЛУЧИЛ, а не то, что
// ОБРАБОТАЛ. Подходит ли он для at-least-once?
func (b *Broker) Subscribe(ctx context.Context, group string, h port.EventHandler) error {
	return todo.Step("08")
}

func (b *Broker) Close() error {
	b.prod.Close()
	return nil
}
