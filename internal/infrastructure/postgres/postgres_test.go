//go:build integration

package postgres_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/s4bb4t/infra-layer-practice/internal/app/port"
	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/contracttest"
	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/postgres"
	"github.com/s4bb4t/infra-layer-practice/internal/testinfra"
)

// One container for the whole package: starting Postgres takes seconds,
// each test takes milliseconds.
var pg *testinfra.Container

func TestMain(m *testing.M) {
	testinfra.Main(m, testinfra.StartPostgres, func(c *testinfra.Container) { pg = c })
}

func newPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, pg.URI)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	// Migrate is idempotent: goose skips applied versions.
	if err := postgres.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	return pool
}

func TestOrderRepositoryContract(t *testing.T) {
	contracttest.RunOrderRepository(t, func(t *testing.T) port.OrderRepository {
		return postgres.NewOrders(newPool(t))
	})
}

func TestTxOutboxContract(t *testing.T) {
	contracttest.RunTxOutbox(t, func(t *testing.T) contracttest.Store {
		pool := newPool(t)
		if _, err := pool.Exec(context.Background(), `TRUNCATE outbox`); err != nil {
			t.Fatal(err)
		}
		ob := postgres.NewOutbox(pool)
		return contracttest.Store{
			Tx:     postgres.NewTxManager(pool),
			Orders: postgres.NewOrders(pool),
			Outbox: ob,
			Reader: ob,
		}
	})
}
