//go:build e2e || chaos

package harness

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/s4bb4t/infra-layer-practice/internal/app/domain"
	"github.com/s4bb4t/infra-layer-practice/internal/app/port"
	"github.com/s4bb4t/infra-layer-practice/internal/app/usecase"
	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/contracttest"
	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/outbox"
	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/system"
)

// Stack names one combination of adapters. Empty Cache means no cache.
type Stack struct {
	Store       string
	Broker      string
	Cache       string
	Projections []string
}

func (s Stack) String() string {
	name := s.Store + "+" + s.Broker
	if s.Cache != "" {
		name += "+" + s.Cache
	}
	return name
}

// App is the whole service wired from real adapters, minus HTTP.
type App struct {
	Store       contracttest.Store
	Broker      contracttest.Broker
	Orders      port.OrderRepository // Store.Orders, possibly wrapped by a cache
	Projections map[string]Projection

	Place  *usecase.PlaceOrder
	Pay    *usecase.PayOrder
	Cancel *usecase.CancelOrder
	Get    *usecase.GetOrder

	Relay *outbox.Relay
	// Ctx lives until the test ends; background workers stop with it.
	Ctx context.Context
}

var Discard = slog.New(slog.NewTextHandler(io.Discard, nil))

// Build wires the stack and starts the outbox relay in the background.
func Build(t *testing.T, s Stack) *App {
	t.Helper()
	inf := Shared()

	sf, ok := stores[s.Store]
	if !ok {
		t.Fatalf("store %q is not registered", s.Store)
	}
	bf, ok := brokers[s.Broker]
	if !ok {
		t.Fatalf("broker %q is not registered", s.Broker)
	}

	// Cleanups run in LIFO order: cancel is registered last below, so it runs
	// first, then the relay is awaited, then adapters close.
	ctx, cancel := context.WithCancel(context.Background())

	app := &App{Ctx: ctx, Projections: map[string]Projection{}}
	app.Store = sf(t, inf)
	app.Broker = bf(t, inf, dbName(t))
	app.Orders = app.Store.Orders
	if s.Cache != "" {
		cf, ok := caches[s.Cache]
		if !ok {
			t.Fatalf("cache %q is not registered", s.Cache)
		}
		app.Orders = cf(t, inf, app.Orders)
	}

	deps := usecase.Deps{Tx: app.Store.Tx, Orders: app.Orders, Outbox: app.Store.Outbox, Clock: system.Clock{}, IDs: system.UUIDv7{}}
	app.Place = usecase.NewPlaceOrder(deps)
	app.Pay = usecase.NewPayOrder(deps)
	app.Cancel = usecase.NewCancelOrder(deps)
	app.Get = usecase.NewGetOrder(app.Orders)

	for _, name := range s.Projections {
		pf, ok := projections[name]
		if !ok {
			t.Fatalf("projection %q is not registered", name)
		}
		p := pf(t, inf)
		if err := app.Broker.Sub.Subscribe(ctx, p.Group, p.Handler); err != nil {
			t.Fatal(err)
		}
		app.Projections[name] = p
	}

	app.Relay = outbox.NewRelay(app.Store.Tx, app.Store.Reader, app.Broker.Pub, Discard)
	var wg sync.WaitGroup
	wg.Go(func() { _ = app.Relay.Run(ctx) })
	t.Cleanup(wg.Wait)
	t.Cleanup(cancel)
	return app
}

// Recorder subscribes a fresh group and remembers every delivered event.
type Recorder struct {
	mu     sync.Mutex
	events []domain.Event
}

func (a *App) Record(t *testing.T) *Recorder {
	t.Helper()
	r := &Recorder{}
	err := a.Broker.Sub.Subscribe(a.Ctx, "recorder-"+suffix(), func(_ context.Context, e domain.Event) error {
		r.mu.Lock()
		r.events = append(r.events, e)
		r.mu.Unlock()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// Events returns a copy of what was delivered so far (duplicates included).
func (r *Recorder) Events() []domain.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]domain.Event(nil), r.events...)
}

// Of returns delivered events of one order, deduplicated by event ID, in delivery order.
func (r *Recorder) Of(id domain.OrderID) []domain.Event {
	seen := map[string]bool{}
	var out []domain.Event
	for _, e := range r.Events() {
		if e.AggregateID == id && !seen[e.ID] {
			seen[e.ID] = true
			out = append(out, e)
		}
	}
	return out
}

// Eventually polls cond until it returns true or timeout passes.
// Read models and brokers are asynchronous: never assert on them once.
func Eventually(t *testing.T, timeout time.Duration, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("not within %s: %s", timeout, msg)
}

func suffix() string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
