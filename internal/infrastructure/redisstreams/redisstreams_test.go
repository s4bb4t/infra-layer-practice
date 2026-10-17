//go:build integration

package redisstreams_test

import (
	"testing"

	goredis "github.com/redis/go-redis/v9"

	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/contracttest"
	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/redisstreams"
	"github.com/s4bb4t/infra-layer-practice/internal/testinfra"
)

var rd *testinfra.Container

func TestMain(m *testing.M) {
	testinfra.Main(m, testinfra.StartRedis, func(c *testinfra.Container) { rd = c })
}

func TestBrokerContract(t *testing.T) {
	contracttest.RunBroker(t, func(t *testing.T) contracttest.Broker {
		rdb := goredis.NewClient(&goredis.Options{Addr: rd.URI})
		t.Cleanup(func() { _ = rdb.Close() })
		b, err := redisstreams.New(rdb, "orders-"+t.Name(), nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = b.Close() })
		return contracttest.Broker{Pub: b, Sub: b}
	})
}
