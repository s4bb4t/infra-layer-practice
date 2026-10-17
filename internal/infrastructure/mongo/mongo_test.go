//go:build integration

package mongo_test

import (
	"context"
	"testing"

	mongodrv "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/s4bb4t/infra-layer-practice/internal/app/port"
	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/contracttest"
	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/mongo"
	"github.com/s4bb4t/infra-layer-practice/internal/testinfra"
)

var mg *testinfra.Container

func TestMain(m *testing.M) {
	testinfra.Main(m, testinfra.StartMongo, func(c *testinfra.Container) { mg = c })
}

func newDB(t *testing.T) (*mongodrv.Client, *mongodrv.Database) {
	t.Helper()
	ctx := context.Background()
	client, err := mongodrv.Connect(options.Client().ApplyURI(mg.URI))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })
	db := client.Database("orders")
	if err := mongo.EnsureIndexes(ctx, db); err != nil {
		t.Fatal(err)
	}
	return client, db
}

func TestOrderRepositoryContract(t *testing.T) {
	contracttest.RunOrderRepository(t, func(t *testing.T) port.OrderRepository {
		_, db := newDB(t)
		return mongo.NewOrders(db)
	})
}

func TestTxOutboxContract(t *testing.T) {
	contracttest.RunTxOutbox(t, func(t *testing.T) contracttest.Store {
		client, db := newDB(t)
		if _, err := db.Collection("outbox").DeleteMany(context.Background(), map[string]any{}); err != nil {
			t.Fatal(err)
		}
		ob := mongo.NewOutbox(db)
		return contracttest.Store{
			Tx:     mongo.NewTxManager(client),
			Orders: mongo.NewOrders(db),
			Outbox: ob,
			Reader: ob,
		}
	})
}
