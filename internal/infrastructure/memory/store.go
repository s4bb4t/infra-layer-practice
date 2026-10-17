// Package memory is the reference in-memory implementation of the storage ports.
//
// It is used in unit tests and as the "obviously correct" baseline: every real
// adapter must pass the same contract tests that this package passes.
package memory

import (
	"context"
	"slices"
	"sync"

	"github.com/s4bb4t/infra-layer-practice/internal/app/domain"
	"github.com/s4bb4t/infra-layer-practice/internal/app/port"
)

var (
	_ port.TxManager       = (*Store)(nil)
	_ port.OrderRepository = (*Store)(nil)
	_ port.Outbox          = (*Store)(nil)
	_ port.OutboxReader    = (*Store)(nil)
)

type outboxRecord struct {
	event     domain.Event
	published bool
}

type state struct {
	orders map[domain.OrderID]domain.Order
	outbox []outboxRecord
}

func (s state) clone() state {
	c := state{
		orders: make(map[domain.OrderID]domain.Order, len(s.orders)),
		outbox: slices.Clone(s.outbox),
	}
	for k, v := range s.orders {
		c.orders[k] = v
	}
	return c
}

// Store implements TxManager, OrderRepository, Outbox and OutboxReader.
//
// Transactions are copy-on-write: Do clones the state, runs fn against the
// clone (found via ctx) and swaps it in on success. writeMu serializes all
// writers, so transactions are effectively SERIALIZABLE.
type Store struct {
	writeMu sync.Mutex   // held by every writer for the whole write / transaction
	mu      sync.RWMutex // protects st
	st      state
}

func NewStore() *Store {
	return &Store{st: state{orders: map[domain.OrderID]domain.Order{}}}
}

type txKey struct{}

func (s *Store) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(*state); ok {
		// Already inside a transaction: join it.
		return fn(ctx)
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	s.mu.RLock()
	tx := s.st.clone()
	s.mu.RUnlock()

	if err := fn(context.WithValue(ctx, txKey{}, &tx)); err != nil {
		return err
	}

	s.mu.Lock()
	s.st = tx
	s.mu.Unlock()
	return nil
}

// read runs f against the transaction state if ctx has one, or the committed state.
func (s *Store) read(ctx context.Context, f func(st *state)) {
	if tx, ok := ctx.Value(txKey{}).(*state); ok {
		f(tx)
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	f(&s.st)
}

// write runs f against the transaction state, or as a single-statement transaction.
func (s *Store) write(ctx context.Context, f func(st *state) error) error {
	if tx, ok := ctx.Value(txKey{}).(*state); ok {
		return f(tx)
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	return f(&s.st)
}

func (s *Store) Create(ctx context.Context, o domain.Order) error {
	return s.write(ctx, func(st *state) error {
		if _, ok := st.orders[o.ID]; ok {
			return domain.ErrAlreadyExists
		}
		st.orders[o.ID] = cloneOrder(o)
		return nil
	})
}

func (s *Store) Get(ctx context.Context, id domain.OrderID) (domain.Order, error) {
	var (
		o  domain.Order
		ok bool
	)
	s.read(ctx, func(st *state) { o, ok = st.orders[id] })
	if !ok {
		return domain.Order{}, domain.ErrNotFound
	}
	return cloneOrder(o), nil
}

func (s *Store) Update(ctx context.Context, o domain.Order) error {
	return s.write(ctx, func(st *state) error {
		cur, ok := st.orders[o.ID]
		if !ok {
			return domain.ErrNotFound
		}
		if cur.Version != o.Version {
			return domain.ErrConflict
		}
		o = cloneOrder(o)
		o.Version++
		st.orders[o.ID] = o
		return nil
	})
}

func (s *Store) ListByCustomer(ctx context.Context, customer domain.CustomerID, limit int) ([]domain.Order, error) {
	var out []domain.Order
	s.read(ctx, func(st *state) {
		for _, o := range st.orders {
			if o.CustomerID == customer {
				out = append(out, cloneOrder(o))
			}
		}
	})
	slices.SortFunc(out, func(a, b domain.Order) int { return b.CreatedAt.Compare(a.CreatedAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *Store) Add(ctx context.Context, events ...domain.Event) error {
	return s.write(ctx, func(st *state) error {
		for _, e := range events {
			st.outbox = append(st.outbox, outboxRecord{event: e})
		}
		return nil
	})
}

func (s *Store) Fetch(ctx context.Context, limit int) ([]domain.Event, error) {
	var out []domain.Event
	s.read(ctx, func(st *state) {
		for _, r := range st.outbox {
			if len(out) == limit {
				break
			}
			if !r.published {
				out = append(out, r.event)
			}
		}
	})
	return out, nil
}

func (s *Store) MarkPublished(ctx context.Context, ids ...string) error {
	return s.write(ctx, func(st *state) error {
		for i := range st.outbox {
			if slices.Contains(ids, st.outbox[i].event.ID) {
				st.outbox[i].published = true
			}
		}
		return nil
	})
}

func cloneOrder(o domain.Order) domain.Order {
	o.Items = slices.Clone(o.Items)
	return o
}
