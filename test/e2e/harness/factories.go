//go:build e2e || chaos

package harness

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
	mongodrv "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/contracttest"
	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/kafka"
	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/memory"
	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/mongo"
	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/mysql"
	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/postgres"
	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/rabbit"
	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/redisstreams"
)

// Factories for adapters that exist in the repo from the start.
func init() {
	RegisterStore("memory", func(t *testing.T, _ *Infra) contracttest.Store {
		s := memory.NewStore()
		return contracttest.Store{Tx: s, Orders: s, Outbox: s, Reader: s}
	})
	RegisterStore("postgres", PostgresStore)
	RegisterStore("mysql", MySQLStore)
	RegisterStore("mongo", MongoStore)

	RegisterBroker("rabbit", RabbitBroker)
	RegisterBroker("redisstreams", RedisStreamsBroker)
	RegisterBroker("kafka", KafkaBroker)
}

// PostgresStore uses a fresh database per test: outbox rows of one test must
// not be relayed by another.
func PostgresStore(t *testing.T, inf *Infra) contracttest.Store {
	return PostgresStoreAt(t, inf.Postgres(t))
}

func PostgresStoreAt(t *testing.T, dsn string) contracttest.Store {
	t.Helper()
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	db := dbName(t)
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+db); err != nil {
		t.Fatal(err)
	}

	pool, err := pgxpool.New(ctx, replaceDB(dsn, db))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := postgres.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	ob := postgres.NewOutbox(pool)
	return contracttest.Store{Tx: postgres.NewTxManager(pool), Orders: postgres.NewOrders(pool), Outbox: ob, Reader: ob}
}

func MySQLStore(t *testing.T, inf *Infra) contracttest.Store {
	t.Helper()
	ctx := context.Background()
	dsn := inf.MySQL(t)
	// The test user owns only the "orders" schema: isolate by truncating instead.
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := mysql.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "TRUNCATE outbox"); err != nil {
		t.Fatal(err)
	}
	ob := mysql.NewOutbox(db)
	return contracttest.Store{Tx: mysql.NewTxManager(db), Orders: mysql.NewOrders(db), Outbox: ob, Reader: ob}
}

func MongoStore(t *testing.T, inf *Infra) contracttest.Store {
	t.Helper()
	client, err := mongodrv.Connect(options.Client().ApplyURI(inf.Mongo(t)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })
	db := client.Database(dbName(t))
	if err := mongo.EnsureIndexes(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	ob := mongo.NewOutbox(db)
	return contracttest.Store{Tx: mongo.NewTxManager(client), Orders: mongo.NewOrders(db), Outbox: ob, Reader: ob}
}

func RabbitBroker(t *testing.T, inf *Infra, topic string) contracttest.Broker {
	t.Helper()
	b, err := rabbit.New(inf.Rabbit(t), topic, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = b.Close() })
	return contracttest.Broker{Pub: b, Sub: b}
}

func RedisStreamsBroker(t *testing.T, inf *Infra, topic string) contracttest.Broker {
	t.Helper()
	rdb := goredis.NewClient(&goredis.Options{Addr: inf.Redis(t)})
	t.Cleanup(func() { _ = rdb.Close() })
	b, err := redisstreams.New(rdb, topic, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = b.Close() })
	return contracttest.Broker{Pub: b, Sub: b}
}

func KafkaBroker(t *testing.T, inf *Infra, topic string) contracttest.Broker {
	t.Helper()
	b, err := kafka.New(context.Background(), []string{inf.Kafka(t)}, topic)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = b.Close() })
	return contracttest.Broker{Pub: b, Sub: b}
}

// dbName turns a test name into a valid, unique database/topic name.
// Postgres limits identifiers to 63 bytes, so the readable part is cut.
func dbName(t *testing.T) string {
	name := strings.ToLower(sanitize.Replace(t.Name()))
	if len(name) > 40 {
		name = name[len(name)-40:]
	}
	return "t_" + strings.Trim(name, "_") + "_" + suffix()
}

var sanitize = strings.NewReplacer("/", "_", "-", "_", " ", "_", "+", "_", ".", "_", ":", "_", "=", "_")

func replaceDB(dsn, db string) string {
	// postgres://user:pass@host:port/orders?sslmode=disable
	i := strings.LastIndex(dsn, "/")
	j := strings.Index(dsn[i:], "?")
	if j < 0 {
		return dsn[:i+1] + db
	}
	return dsn[:i+1] + db + dsn[i+j:]
}
