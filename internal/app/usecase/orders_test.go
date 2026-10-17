package usecase_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/s4bb4t/infra-layer-practice/internal/app/domain"
	"github.com/s4bb4t/infra-layer-practice/internal/app/port"
	"github.com/s4bb4t/infra-layer-practice/internal/app/usecase"
	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/memory"
)

var now = time.Date(2026, 5, 6, 7, 8, 9, 0, time.UTC)

type fixedClock struct{}

func (fixedClock) Now() time.Time { return now }

type seqIDs struct{ n int }

func (s *seqIDs) NewID() string { s.n++; return fmt.Sprintf("id-%d", s.n) }

// failingRepo / failingOutbox wrap the memory store and inject errors:
// unit tests cover every branch without any container.
type failingRepo struct {
	port.OrderRepository
	createErr, getErr, updateErr error
}

func (r failingRepo) Create(ctx context.Context, o domain.Order) error {
	if r.createErr != nil {
		return r.createErr
	}
	return r.OrderRepository.Create(ctx, o)
}

func (r failingRepo) Get(ctx context.Context, id domain.OrderID) (domain.Order, error) {
	if r.getErr != nil {
		return domain.Order{}, r.getErr
	}
	return r.OrderRepository.Get(ctx, id)
}

func (r failingRepo) Update(ctx context.Context, o domain.Order) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	return r.OrderRepository.Update(ctx, o)
}

type failingOutbox struct{ err error }

func (f failingOutbox) Add(context.Context, ...domain.Event) error { return f.err }

type env struct {
	store *memory.Store
	deps  usecase.Deps
}

func newEnv() *env {
	s := memory.NewStore()
	return &env{store: s, deps: usecase.Deps{Tx: s, Orders: s, Outbox: s, Clock: fixedClock{}, IDs: &seqIDs{}}}
}

func (e *env) outbox(t *testing.T) []domain.Event {
	t.Helper()
	events, err := e.store.Fetch(context.Background(), 100)
	if err != nil {
		t.Fatal(err)
	}
	return events
}

var items = []domain.Item{{SKU: "a", Qty: 1, PriceCents: 100}}

func TestPlaceOrder(t *testing.T) {
	ctx := context.Background()

	t.Run("stores order and OrderPlaced event", func(t *testing.T) {
		e := newEnv()
		o, err := usecase.NewPlaceOrder(e.deps).Execute(ctx, usecase.PlaceOrderInput{CustomerID: "c", Items: items})
		if err != nil {
			t.Fatal(err)
		}
		if o.ID != "id-1" || o.Status != domain.StatusCreated || !o.CreatedAt.Equal(now) {
			t.Fatalf("unexpected order %+v", o)
		}
		if _, err := e.store.Get(ctx, o.ID); err != nil {
			t.Fatal(err)
		}

		events := e.outbox(t)
		if len(events) != 1 || events[0].Type != domain.EventOrderPlaced || events[0].AggregateID != o.ID {
			t.Fatalf("outbox = %+v", events)
		}
		var p domain.OrderPlacedPayload
		if err := json.Unmarshal(events[0].Payload, &p); err != nil || p.Order.TotalCents != 100 {
			t.Fatalf("payload = %s, err = %v", events[0].Payload, err)
		}
	})

	t.Run("invalid input touches nothing", func(t *testing.T) {
		e := newEnv()
		_, err := usecase.NewPlaceOrder(e.deps).Execute(ctx, usecase.PlaceOrderInput{CustomerID: "c"})
		if !errors.Is(err, domain.ErrInvalidOrder) {
			t.Fatalf("err = %v", err)
		}
		if len(e.outbox(t)) != 0 {
			t.Fatal("outbox must be empty")
		}
	})

	t.Run("repository failure is returned, no event", func(t *testing.T) {
		e := newEnv()
		boom := errors.New("db down")
		e.deps.Orders = failingRepo{OrderRepository: e.store, createErr: boom}

		_, err := usecase.NewPlaceOrder(e.deps).Execute(ctx, usecase.PlaceOrderInput{CustomerID: "c", Items: items})
		if !errors.Is(err, boom) {
			t.Fatalf("err = %v", err)
		}
		if len(e.outbox(t)) != 0 {
			t.Fatal("outbox must be empty")
		}
	})

	t.Run("outbox failure rolls back the order", func(t *testing.T) {
		e := newEnv()
		boom := errors.New("outbox down")
		e.deps.Outbox = failingOutbox{err: boom}

		_, err := usecase.NewPlaceOrder(e.deps).Execute(ctx, usecase.PlaceOrderInput{CustomerID: "c", Items: items})
		if !errors.Is(err, boom) {
			t.Fatalf("err = %v", err)
		}
		if _, err := e.store.Get(ctx, "id-1"); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("order must be rolled back, got err = %v", err)
		}
	})
}

func TestPayAndCancel(t *testing.T) {
	ctx := context.Background()

	place := func(t *testing.T, e *env) domain.Order {
		t.Helper()
		o, err := usecase.NewPlaceOrder(e.deps).Execute(ctx, usecase.PlaceOrderInput{CustomerID: "c", Items: items})
		if err != nil {
			t.Fatal(err)
		}
		return o
	}

	t.Run("pay updates status, version and records OrderPaid", func(t *testing.T) {
		e := newEnv()
		o := place(t, e)

		paid, err := usecase.NewPayOrder(e.deps).Execute(ctx, o.ID)
		if err != nil {
			t.Fatal(err)
		}
		if paid.Status != domain.StatusPaid || paid.Version != 2 {
			t.Fatalf("got %+v", paid)
		}
		stored, _ := e.store.Get(ctx, o.ID)
		if stored.Version != paid.Version {
			t.Fatalf("returned version %d, stored %d", paid.Version, stored.Version)
		}
		events := e.outbox(t)
		if len(events) != 2 || events[1].Type != domain.EventOrderPaid || events[1].Version != 2 {
			t.Fatalf("outbox = %+v", events)
		}
	})

	t.Run("cancel after pay", func(t *testing.T) {
		e := newEnv()
		o := place(t, e)
		if _, err := usecase.NewPayOrder(e.deps).Execute(ctx, o.ID); err != nil {
			t.Fatal(err)
		}
		c, err := usecase.NewCancelOrder(e.deps).Execute(ctx, o.ID)
		if err != nil {
			t.Fatal(err)
		}
		if c.Status != domain.StatusCancelled || c.Version != 3 {
			t.Fatalf("got %+v", c)
		}
	})

	t.Run("invalid transition records nothing", func(t *testing.T) {
		e := newEnv()
		o := place(t, e)
		if _, err := usecase.NewCancelOrder(e.deps).Execute(ctx, o.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := usecase.NewPayOrder(e.deps).Execute(ctx, o.ID); !errors.Is(err, domain.ErrInvalidTransition) {
			t.Fatalf("err = %v", err)
		}
		if n := len(e.outbox(t)); n != 2 {
			t.Fatalf("outbox has %d events, want 2", n)
		}
	})

	t.Run("missing order", func(t *testing.T) {
		e := newEnv()
		if _, err := usecase.NewPayOrder(e.deps).Execute(ctx, "nope"); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("conflict is returned, no event", func(t *testing.T) {
		e := newEnv()
		o := place(t, e)
		e.deps.Orders = failingRepo{OrderRepository: e.store, updateErr: domain.ErrConflict}

		if _, err := usecase.NewPayOrder(e.deps).Execute(ctx, o.ID); !errors.Is(err, domain.ErrConflict) {
			t.Fatalf("err = %v", err)
		}
		if n := len(e.outbox(t)); n != 1 {
			t.Fatalf("outbox has %d events, want 1", n)
		}
	})

	t.Run("outbox failure rolls back the status change", func(t *testing.T) {
		e := newEnv()
		o := place(t, e)
		e.deps.Outbox = failingOutbox{err: errors.New("outbox down")}

		if _, err := usecase.NewPayOrder(e.deps).Execute(ctx, o.ID); err == nil {
			t.Fatal("want error")
		}
		stored, _ := e.store.Get(ctx, o.ID)
		if stored.Status != domain.StatusCreated {
			t.Fatalf("status = %s, want created", stored.Status)
		}
	})
}
