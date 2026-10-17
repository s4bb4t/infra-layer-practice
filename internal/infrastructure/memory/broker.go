package memory

import (
	"context"
	"sync"
	"time"

	"github.com/s4bb4t/infra-layer-practice/internal/app/domain"
	"github.com/s4bb4t/infra-layer-practice/internal/app/port"
)

var (
	_ port.EventPublisher  = (*Broker)(nil)
	_ port.EventSubscriber = (*Broker)(nil)
)

// RedeliveryDelay is how long a failed event waits before it is handed out again.
const RedeliveryDelay = 10 * time.Millisecond

// Broker is an in-process broker with consumer groups and at-least-once delivery.
//
// Each group has its own queue; publish copies the event into every existing
// group queue (fan-out), and subscribers of the same group compete for it.
// Events published before a group exists are not delivered to that group.
type Broker struct {
	mu     sync.Mutex
	groups map[string]chan domain.Event
}

func NewBroker() *Broker {
	return &Broker{groups: map[string]chan domain.Event{}}
}

func (b *Broker) Publish(ctx context.Context, events ...domain.Event) error {
	b.mu.Lock()
	queues := make([]chan domain.Event, 0, len(b.groups))
	for _, q := range b.groups {
		queues = append(queues, q)
	}
	b.mu.Unlock()

	for _, e := range events {
		for _, q := range queues {
			select {
			case q <- e:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return nil
}

func (b *Broker) Subscribe(ctx context.Context, group string, h port.EventHandler) error {
	b.mu.Lock()
	q, ok := b.groups[group]
	if !ok {
		q = make(chan domain.Event, 4096)
		b.groups[group] = q
	}
	b.mu.Unlock()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case e := <-q:
				if err := h(ctx, e); err != nil {
					go func() {
						select {
						case <-time.After(RedeliveryDelay):
							q <- e
						case <-ctx.Done():
						}
					}()
				}
			}
		}
	}()
	return nil
}
