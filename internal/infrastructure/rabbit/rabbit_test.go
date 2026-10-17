//go:build integration

package rabbit_test

import (
	"testing"

	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/contracttest"
	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/rabbit"
	"github.com/s4bb4t/infra-layer-practice/internal/testinfra"
)

var rmq *testinfra.Container

func TestMain(m *testing.M) {
	testinfra.Main(m, testinfra.StartRabbit, func(c *testinfra.Container) { rmq = c })
}

func TestBrokerContract(t *testing.T) {
	contracttest.RunBroker(t, func(t *testing.T) contracttest.Broker {
		// A fresh topic per subtest: a fresh exchange, no leftovers.
		b, err := rabbit.New(rmq.URI, "orders-"+t.Name(), nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = b.Close() })
		return contracttest.Broker{Pub: b, Sub: b}
	})
}
