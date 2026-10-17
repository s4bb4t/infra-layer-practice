package contracttest

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/s4bb4t/infra-layer-practice/internal/app/domain"
	"github.com/s4bb4t/infra-layer-practice/internal/app/port"
)

// RunOrderRepository checks that repo behaves as port.OrderRepository promises.
//
// newRepo is called once per subtest. It may return the same underlying
// database every time: tests use unique ids and customers.
func RunOrderRepository(t *testing.T, newRawRepo func(t *testing.T) port.OrderRepository) {
	ctx := context.Background()
	newRepo := func(t *testing.T) port.OrderRepository { return repoGuard{t: t, r: newRawRepo(t)} }

	t.Run("create then get returns the same order", func(t *testing.T) {
		repo := newRepo(t)
		o := newOrder(t, domain.CustomerID(uniq("c")), baseTime)

		if err := repo.Create(ctx, o); err != nil {
			t.Fatal(err)
		}
		got, err := repo.Get(ctx, o.ID)
		if err != nil {
			t.Fatal(err)
		}
		assertOrder(t, got, o)
	})

	t.Run("create with existing id fails with ErrAlreadyExists", func(t *testing.T) {
		repo := newRepo(t)
		o := newOrder(t, domain.CustomerID(uniq("c")), baseTime)

		if err := repo.Create(ctx, o); err != nil {
			t.Fatal(err)
		}
		if err := repo.Create(ctx, o); !errors.Is(err, domain.ErrAlreadyExists) {
			t.Fatalf("err = %v, want ErrAlreadyExists", err)
		}
	})

	t.Run("get missing fails with ErrNotFound", func(t *testing.T) {
		repo := newRepo(t)
		if _, err := repo.Get(ctx, domain.OrderID(uniq("missing"))); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("update saves changes and bumps version", func(t *testing.T) {
		repo := newRepo(t)
		o := newOrder(t, domain.CustomerID(uniq("c")), baseTime)
		if err := repo.Create(ctx, o); err != nil {
			t.Fatal(err)
		}

		if err := o.Pay(); err != nil {
			t.Fatal(err)
		}
		if err := repo.Update(ctx, o); err != nil {
			t.Fatal(err)
		}

		got, err := repo.Get(ctx, o.ID)
		if err != nil {
			t.Fatal(err)
		}
		o.Version++
		assertOrder(t, got, o)
	})

	t.Run("update with stale version fails with ErrConflict", func(t *testing.T) {
		repo := newRepo(t)
		o := newOrder(t, domain.CustomerID(uniq("c")), baseTime)
		if err := repo.Create(ctx, o); err != nil {
			t.Fatal(err)
		}
		if err := repo.Update(ctx, o); err != nil {
			t.Fatal(err)
		}

		// o still has Version 1, storage has 2.
		if err := repo.Update(ctx, o); !errors.Is(err, domain.ErrConflict) {
			t.Fatalf("err = %v, want ErrConflict", err)
		}
	})

	t.Run("update missing fails with ErrNotFound", func(t *testing.T) {
		repo := newRepo(t)
		o := newOrder(t, domain.CustomerID(uniq("c")), baseTime)
		if err := repo.Update(ctx, o); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("concurrent updates of one version: exactly one wins", func(t *testing.T) {
		raw := newRawRepo(t)
		repo := repoGuard{t: t, r: raw}
		o := newOrder(t, domain.CustomerID(uniq("c")), baseTime)
		if err := repo.Create(ctx, o); err != nil {
			t.Fatal(err)
		}

		const n = 10
		errs := make([]error, n)
		var wg sync.WaitGroup
		for i := range n {
			// raw, not guarded: t.Skip must not be called from other goroutines.
			wg.Go(func() { errs[i] = raw.Update(ctx, o) })
		}
		wg.Wait()

		var ok, conflicts int
		for _, err := range errs {
			_ = skipTODO(t, err)
			switch {
			case err == nil:
				ok++
			case errors.Is(err, domain.ErrConflict):
				conflicts++
			default:
				t.Fatalf("unexpected error: %v", err)
			}
		}
		if ok != 1 || conflicts != n-1 {
			t.Fatalf("ok = %d, conflicts = %d; want 1 and %d", ok, conflicts, n-1)
		}
	})

	t.Run("list by customer: newest first, limited, only this customer", func(t *testing.T) {
		repo := newRepo(t)
		customer := domain.CustomerID(uniq("c"))

		var orders []domain.Order
		for i := range 4 {
			o := newOrder(t, customer, baseTime.Add(time.Duration(i)*time.Minute))
			orders = append(orders, o)
			if err := repo.Create(ctx, o); err != nil {
				t.Fatal(err)
			}
		}
		if err := repo.Create(ctx, newOrder(t, domain.CustomerID(uniq("other")), baseTime)); err != nil {
			t.Fatal(err)
		}

		got, err := repo.ListByCustomer(ctx, customer, 3)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 3 {
			t.Fatalf("len = %d, want 3", len(got))
		}
		for i, want := range []domain.Order{orders[3], orders[2], orders[1]} {
			assertOrder(t, got[i], want)
		}
	})

	t.Run("list by unknown customer is empty, not an error", func(t *testing.T) {
		repo := newRepo(t)
		got, err := repo.ListByCustomer(ctx, domain.CustomerID(uniq("nobody")), 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Fatalf("got %d orders, want 0", len(got))
		}
	})
}
