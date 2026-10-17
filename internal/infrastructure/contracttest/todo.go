package contracttest

import (
	"context"
	"errors"
	"testing"

	"github.com/s4bb4t/infra-layer-practice/internal/app/domain"
	"github.com/s4bb4t/infra-layer-practice/internal/app/port"
	"github.com/s4bb4t/infra-layer-practice/internal/todo"
)

// The guards below wrap an implementation and turn todo.ErrNotImplemented into
// t.Skip. They must only be called from the test goroutine.

func skipTODO(t *testing.T, err error) error {
	t.Helper()
	if errors.Is(err, todo.ErrNotImplemented) {
		t.Skip(err.Error())
	}
	return err
}

type repoGuard struct {
	t *testing.T
	r port.OrderRepository
}

func (g repoGuard) Create(ctx context.Context, o domain.Order) error {
	return skipTODO(g.t, g.r.Create(ctx, o))
}

func (g repoGuard) Get(ctx context.Context, id domain.OrderID) (domain.Order, error) {
	o, err := g.r.Get(ctx, id)
	return o, skipTODO(g.t, err)
}

func (g repoGuard) Update(ctx context.Context, o domain.Order) error {
	return skipTODO(g.t, g.r.Update(ctx, o))
}

func (g repoGuard) ListByCustomer(ctx context.Context, c domain.CustomerID, limit int) ([]domain.Order, error) {
	out, err := g.r.ListByCustomer(ctx, c, limit)
	return out, skipTODO(g.t, err)
}

type txGuard struct {
	t  *testing.T
	tx port.TxManager
}

func (g txGuard) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	return skipTODO(g.t, g.tx.Do(ctx, fn))
}

type outboxGuard struct {
	t *testing.T
	o port.Outbox
	r port.OutboxReader
}

func (g outboxGuard) Add(ctx context.Context, events ...domain.Event) error {
	return skipTODO(g.t, g.o.Add(ctx, events...))
}

func (g outboxGuard) Fetch(ctx context.Context, limit int) ([]domain.Event, error) {
	out, err := g.r.Fetch(ctx, limit)
	return out, skipTODO(g.t, err)
}

func (g outboxGuard) MarkPublished(ctx context.Context, ids ...string) error {
	return skipTODO(g.t, g.r.MarkPublished(ctx, ids...))
}

type pubGuard struct {
	t *testing.T
	p port.EventPublisher
}

func (g pubGuard) Publish(ctx context.Context, events ...domain.Event) error {
	return skipTODO(g.t, g.p.Publish(ctx, events...))
}

type subGuard struct {
	t *testing.T
	s port.EventSubscriber
}

func (g subGuard) Subscribe(ctx context.Context, group string, h port.EventHandler) error {
	return skipTODO(g.t, g.s.Subscribe(ctx, group, h))
}
