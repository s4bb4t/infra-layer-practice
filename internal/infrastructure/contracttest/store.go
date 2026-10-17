package contracttest

import (
	"context"
	"errors"
	"testing"

	"github.com/s4bb4t/infra-layer-practice/internal/app/domain"
	"github.com/s4bb4t/infra-layer-practice/internal/app/port"
)

// Store is the set of ports one storage technology must provide together:
// the order repository and the outbox share one transaction.
type Store struct {
	Tx     port.TxManager
	Orders port.OrderRepository
	Outbox port.Outbox
	Reader port.OutboxReader
}

// RunTxOutbox checks transactions across OrderRepository and Outbox.
//
// newStore is called once per subtest and MUST return a store with an empty
// outbox (truncate the table / drop the collection).
func RunTxOutbox(t *testing.T, newRawStore func(t *testing.T) Store) {
	ctx := context.Background()
	newStore := func(t *testing.T) Store {
		s := newRawStore(t)
		ob := outboxGuard{t: t, o: s.Outbox, r: s.Reader}
		return Store{Tx: txGuard{t: t, tx: s.Tx}, Orders: repoGuard{t: t, r: s.Orders}, Outbox: ob, Reader: ob}
	}

	t.Run("commit persists order and events together", func(t *testing.T) {
		s := newStore(t)
		o := newOrder(t, domain.CustomerID(uniq("c")), baseTime)
		e := newEvent(t, o)

		err := s.Tx.Do(ctx, func(ctx context.Context) error {
			if err := s.Orders.Create(ctx, o); err != nil {
				return err
			}
			return s.Outbox.Add(ctx, e)
		})
		if err != nil {
			t.Fatal(err)
		}

		got, err := s.Orders.Get(ctx, o.ID)
		if err != nil {
			t.Fatal(err)
		}
		assertOrder(t, got, o)

		events, err := s.Reader.Fetch(ctx, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(events) != 1 {
			t.Fatalf("outbox has %d events, want 1", len(events))
		}
		assertEvent(t, events[0], e)
	})

	t.Run("error in fn rolls back order and events and is returned", func(t *testing.T) {
		s := newStore(t)
		o := newOrder(t, domain.CustomerID(uniq("c")), baseTime)
		boom := errors.New("boom")

		err := s.Tx.Do(ctx, func(ctx context.Context) error {
			if err := s.Orders.Create(ctx, o); err != nil {
				return err
			}
			if err := s.Outbox.Add(ctx, newEvent(t, o)); err != nil {
				return err
			}
			return boom
		})
		if !errors.Is(err, boom) {
			t.Fatalf("err = %v, want boom", err)
		}

		if _, err := s.Orders.Get(ctx, o.ID); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("order survived rollback: err = %v", err)
		}
		events, err := s.Reader.Fetch(ctx, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(events) != 0 {
			t.Fatalf("outbox has %d events after rollback", len(events))
		}
	})

	t.Run("writes are visible inside the transaction", func(t *testing.T) {
		s := newStore(t)
		o := newOrder(t, domain.CustomerID(uniq("c")), baseTime)

		err := s.Tx.Do(ctx, func(ctx context.Context) error {
			if err := s.Orders.Create(ctx, o); err != nil {
				return err
			}
			got, err := s.Orders.Get(ctx, o.ID)
			if err != nil {
				return err
			}
			assertOrder(t, got, o)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("fetch returns oldest first, respects limit, skips published", func(t *testing.T) {
		s := newStore(t)
		o := newOrder(t, domain.CustomerID(uniq("c")), baseTime)

		var events []domain.Event
		for range 5 {
			e := newEvent(t, o)
			events = append(events, e)
			// Separate transactions: insertion order must be preserved across them.
			if err := s.Tx.Do(ctx, func(ctx context.Context) error { return s.Outbox.Add(ctx, e) }); err != nil {
				t.Fatal(err)
			}
		}

		got, err := s.Reader.Fetch(ctx, 3)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 3 {
			t.Fatalf("fetched %d, want 3", len(got))
		}
		for i := range got {
			assertEvent(t, got[i], events[i])
		}

		if err := s.Reader.MarkPublished(ctx, events[0].ID, events[1].ID); err != nil {
			t.Fatal(err)
		}
		got, err = s.Reader.Fetch(ctx, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 3 {
			t.Fatalf("after mark: fetched %d, want 3", len(got))
		}
		for i := range got {
			assertEvent(t, got[i], events[i+2])
		}
	})

	t.Run("several events in one Add keep their order", func(t *testing.T) {
		s := newStore(t)
		o := newOrder(t, domain.CustomerID(uniq("c")), baseTime)
		e1, e2, e3 := newEvent(t, o), newEvent(t, o), newEvent(t, o)

		if err := s.Tx.Do(ctx, func(ctx context.Context) error { return s.Outbox.Add(ctx, e1, e2, e3) }); err != nil {
			t.Fatal(err)
		}
		got, err := s.Reader.Fetch(ctx, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 3 {
			t.Fatalf("fetched %d, want 3", len(got))
		}
		for i, want := range []domain.Event{e1, e2, e3} {
			assertEvent(t, got[i], want)
		}
	})

	t.Run("conflict inside a transaction rolls everything back", func(t *testing.T) {
		s := newStore(t)
		o := newOrder(t, domain.CustomerID(uniq("c")), baseTime)
		if err := s.Orders.Create(ctx, o); err != nil {
			t.Fatal(err)
		}
		if err := s.Orders.Update(ctx, o); err != nil {
			t.Fatal(err)
		}

		err := s.Tx.Do(ctx, func(ctx context.Context) error {
			if err := s.Outbox.Add(ctx, newEvent(t, o)); err != nil {
				return err
			}
			return s.Orders.Update(ctx, o) // stale version
		})
		if !errors.Is(err, domain.ErrConflict) {
			t.Fatalf("err = %v, want ErrConflict", err)
		}
		events, err := s.Reader.Fetch(ctx, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(events) != 0 {
			t.Fatalf("outbox has %d events after rollback", len(events))
		}
	})
}
