// Package port declares what the application needs from the outside world.
//
// Every interface here is implemented in internal/infrastructure/<tech>.
// Use cases depend only on these interfaces, so swapping Postgres for Mongo
// or Kafka for NATS never touches internal/app.
package port

import (
	"context"
	"time"

	"github.com/s4bb4t/infra-layer-practice/internal/app/domain"
)

// TxManager runs fn inside a transaction.
//
// The transaction travels inside ctx: repository and outbox adapters of the
// same technology pick it up from the context passed to them. If fn returns an
// error, everything written through that ctx is rolled back.
type TxManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

// OrderRepository is the source of truth for orders.
type OrderRepository interface {
	// Create stores a new order. Returns domain.ErrAlreadyExists if the ID is taken.
	Create(ctx context.Context, o domain.Order) error
	// Get returns domain.ErrNotFound if there is no such order.
	Get(ctx context.Context, id domain.OrderID) (domain.Order, error)
	// Update saves o only if the stored version equals o.Version and stores
	// o.Version+1. Returns domain.ErrConflict on version mismatch and
	// domain.ErrNotFound if there is no such order.
	Update(ctx context.Context, o domain.Order) error
	// ListByCustomer returns at most limit orders of the customer, newest first.
	ListByCustomer(ctx context.Context, customer domain.CustomerID, limit int) ([]domain.Order, error)
}

// Outbox stores events in the same transaction as the order change.
type Outbox interface {
	Add(ctx context.Context, events ...domain.Event) error
}

// OutboxReader is used by the relay that moves events from the outbox to a broker.
type OutboxReader interface {
	// Fetch returns up to limit unpublished events, oldest first.
	Fetch(ctx context.Context, limit int) ([]domain.Event, error)
	// MarkPublished marks events as delivered to the broker.
	MarkPublished(ctx context.Context, ids ...string) error
}

// EventPublisher sends events to a broker.
type EventPublisher interface {
	Publish(ctx context.Context, events ...domain.Event) error
}

// EventHandler processes one event. Returning an error asks the broker to
// redeliver it later, so handlers must be idempotent.
type EventHandler func(ctx context.Context, e domain.Event) error

// EventSubscriber consumes events as a named group.
//
// Every group receives every event (fan-out between groups); inside a group
// each event is handled by one consumer (competing consumers).
type EventSubscriber interface {
	// Subscribe returns once the subscription is ready: any event published
	// after Subscribe returns is guaranteed to reach the group. Consumption
	// runs in the background until ctx is cancelled.
	Subscribe(ctx context.Context, group string, h EventHandler) error
}

// Cache is a byte-oriented key-value cache.
type Cache interface {
	// Get returns ok=false on a miss; err is reserved for cache failures.
	Get(ctx context.Context, key string) (val []byte, ok bool, err error)
	Set(ctx context.Context, key string, val []byte, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
}

// Clock and IDGenerator make use cases deterministic in tests.
type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID() string
}
