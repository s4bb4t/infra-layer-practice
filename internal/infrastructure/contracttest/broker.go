package contracttest

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/s4bb4t/infra-layer-practice/internal/app/domain"
	"github.com/s4bb4t/infra-layer-practice/internal/app/port"
)

// Broker is a publisher/subscriber pair of one technology.
type Broker struct {
	Pub port.EventPublisher
	Sub port.EventSubscriber
}

// BrokerTimeout bounds how long a test waits for deliveries. Real brokers in
// containers can take seconds to rebalance a consumer group.
var BrokerTimeout = 60 * time.Second

// RunBroker checks at-least-once delivery semantics.
//
// newBroker is called once per subtest and must return an isolated
// topic/stream/exchange, so that subtests do not see each other's events.
func RunBroker(t *testing.T, newRawBroker func(t *testing.T) Broker) {
	newBroker := func(t *testing.T) Broker {
		b := newRawBroker(t)
		return Broker{Pub: pubGuard{t: t, p: b.Pub}, Sub: subGuard{t: t, s: b.Sub}}
	}
	t.Run("group receives every event published after subscribe", func(t *testing.T) {
		b := newBroker(t)
		ctx := testCtx(t)
		rec := newRecorder()

		if err := b.Sub.Subscribe(ctx, uniq("g"), rec.handle(nil)); err != nil {
			t.Fatal(err)
		}
		events := publishN(t, ctx, b.Pub, 10)
		rec.waitAll(t, events)
	})

	t.Run("event fields survive the round trip", func(t *testing.T) {
		b := newBroker(t)
		ctx := testCtx(t)
		rec := newRecorder()

		if err := b.Sub.Subscribe(ctx, uniq("g"), rec.handle(nil)); err != nil {
			t.Fatal(err)
		}
		events := publishN(t, ctx, b.Pub, 1)
		rec.waitAll(t, events)
		assertEvent(t, rec.get(events[0].ID), events[0])
	})

	t.Run("failed handler gets the event again", func(t *testing.T) {
		b := newBroker(t)
		ctx := testCtx(t)
		rec := newRecorder()

		var mu sync.Mutex
		failed := map[string]bool{}
		failOnce := func(e domain.Event) bool {
			mu.Lock()
			defer mu.Unlock()
			if failed[e.ID] {
				return false
			}
			failed[e.ID] = true
			return true
		}

		if err := b.Sub.Subscribe(ctx, uniq("g"), rec.handle(failOnce)); err != nil {
			t.Fatal(err)
		}
		events := publishN(t, ctx, b.Pub, 3)
		rec.waitAll(t, events)
	})

	t.Run("two groups each receive every event", func(t *testing.T) {
		b := newBroker(t)
		ctx := testCtx(t)
		rec1, rec2 := newRecorder(), newRecorder()

		if err := b.Sub.Subscribe(ctx, uniq("g1"), rec1.handle(nil)); err != nil {
			t.Fatal(err)
		}
		if err := b.Sub.Subscribe(ctx, uniq("g2"), rec2.handle(nil)); err != nil {
			t.Fatal(err)
		}
		events := publishN(t, ctx, b.Pub, 5)
		rec1.waitAll(t, events)
		rec2.waitAll(t, events)
	})

	t.Run("consumers of one group share the work", func(t *testing.T) {
		b := newBroker(t)
		ctx := testCtx(t)
		rec := newRecorder() // shared: the group as a whole must see everything
		group := uniq("g")

		for range 2 {
			if err := b.Sub.Subscribe(ctx, group, rec.handle(nil)); err != nil {
				t.Fatal(err)
			}
		}
		events := publishN(t, ctx, b.Pub, 10)
		rec.waitAll(t, events)
	})
}

func testCtx(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), BrokerTimeout)
	t.Cleanup(cancel)
	return ctx
}

func publishN(t *testing.T, ctx context.Context, pub port.EventPublisher, n int) []domain.Event {
	t.Helper()
	o := newOrder(t, domain.CustomerID(uniq("c")), baseTime)
	events := make([]domain.Event, n)
	for i := range events {
		events[i] = newEvent(t, o)
	}
	if err := pub.Publish(ctx, events...); err != nil {
		t.Fatal(err)
	}
	return events
}

// recorder collects successfully handled events; duplicates are allowed.
type recorder struct {
	mu   sync.Mutex
	seen map[string]domain.Event
	ch   chan struct{}
}

func newRecorder() *recorder {
	return &recorder{seen: map[string]domain.Event{}, ch: make(chan struct{}, 1)}
}

func (r *recorder) handle(fail func(domain.Event) bool) port.EventHandler {
	return func(_ context.Context, e domain.Event) error {
		if fail != nil && fail(e) {
			return errTransient
		}
		r.mu.Lock()
		r.seen[e.ID] = e
		r.mu.Unlock()
		select {
		case r.ch <- struct{}{}:
		default:
		}
		return nil
	}
}

func (r *recorder) get(id string) domain.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.seen[id]
}

func (r *recorder) waitAll(t *testing.T, events []domain.Event) {
	t.Helper()
	deadline := time.After(BrokerTimeout)
	for {
		r.mu.Lock()
		missing := 0
		for _, e := range events {
			if _, ok := r.seen[e.ID]; !ok {
				missing++
			}
		}
		r.mu.Unlock()
		if missing == 0 {
			return
		}
		select {
		case <-r.ch:
		case <-time.After(100 * time.Millisecond):
		case <-deadline:
			t.Fatalf("%d of %d events not delivered within %s", missing, len(events), BrokerTimeout)
		}
	}
}

type transientError struct{}

func (transientError) Error() string { return "transient handler failure" }

var errTransient = transientError{}
