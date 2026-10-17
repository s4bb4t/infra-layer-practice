//go:build integration

package mysql_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/go-sql-driver/mysql"

	"github.com/s4bb4t/infra-layer-practice/internal/app/port"
	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/contracttest"
	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/mysql"
	"github.com/s4bb4t/infra-layer-practice/internal/testinfra"
)

var my *testinfra.Container

func TestMain(m *testing.M) {
	testinfra.Main(m, testinfra.StartMySQL, func(c *testinfra.Container) { my = c })
}

func newDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("mysql", my.URI)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := mysql.Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestOrderRepositoryContract(t *testing.T) {
	contracttest.RunOrderRepository(t, func(t *testing.T) port.OrderRepository {
		return mysql.NewOrders(newDB(t))
	})
}

func TestTxOutboxContract(t *testing.T) {
	contracttest.RunTxOutbox(t, func(t *testing.T) contracttest.Store {
		db := newDB(t)
		if _, err := db.ExecContext(context.Background(), `TRUNCATE outbox`); err != nil {
			t.Fatal(err)
		}
		ob := mysql.NewOutbox(db)
		return contracttest.Store{
			Tx:     mysql.NewTxManager(db),
			Orders: mysql.NewOrders(db),
			Outbox: ob,
			Reader: ob,
		}
	})
}
