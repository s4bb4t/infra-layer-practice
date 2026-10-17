//go:build integration

package kafka_test

import (
	"context"
	"strings"
	"testing"

	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/contracttest"
	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/kafka"
	"github.com/s4bb4t/infra-layer-practice/internal/testinfra"
)

var kf *testinfra.Container

func TestMain(m *testing.M) {
	testinfra.Main(m, testinfra.StartKafka, func(c *testinfra.Container) { kf = c })
}

func TestBrokerContract(t *testing.T) {
	contracttest.RunBroker(t, func(t *testing.T) contracttest.Broker {
		// Kafka topic names allow [a-zA-Z0-9._-] only.
		topic := "orders." + strings.NewReplacer("/", ".", " ", "_").Replace(t.Name())
		b, err := kafka.New(context.Background(), []string{kf.URI}, topic)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = b.Close() })
		return contracttest.Broker{Pub: b, Sub: b}
	})
}
