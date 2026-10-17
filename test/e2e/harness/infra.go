//go:build e2e || chaos

// Package harness assembles the whole application from real adapters for
// end-to-end tests: the same use cases on every combination of storage and broker.
package harness

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/s4bb4t/infra-layer-practice/internal/testinfra"
)

// Infra starts each container lazily, once per test binary, and stops all of
// them in Main. A matrix of 3 stores x 4 brokers needs 7 containers, not 12x2.
type Infra struct {
	mu         sync.Mutex
	containers map[string]*testinfra.Container
}

var shared = &Infra{containers: map[string]*testinfra.Container{}}

// Main is the TestMain body for e2e packages.
func Main(m *testing.M) {
	code := m.Run()
	for _, c := range shared.containers {
		_ = c.Terminate(context.Background())
	}
	os.Exit(code)
}

func Shared() *Infra { return shared }

func (inf *Infra) get(t *testing.T, name string, start func(context.Context) (*testinfra.Container, error)) string {
	t.Helper()
	inf.mu.Lock()
	defer inf.mu.Unlock()
	if c, ok := inf.containers[name]; ok {
		return c.URI
	}
	c, err := start(context.Background())
	if err != nil {
		t.Fatalf("start %s: %v", name, err)
	}
	inf.containers[name] = c
	return c.URI
}

// Container returns the started container (for chaos tests: stop/start it).
func (inf *Infra) Container(name string) (*testinfra.Container, error) {
	inf.mu.Lock()
	defer inf.mu.Unlock()
	c, ok := inf.containers[name]
	if !ok {
		return nil, fmt.Errorf("container %s is not started", name)
	}
	return c, nil
}

func (inf *Infra) Postgres(t *testing.T) string {
	return inf.get(t, "postgres", testinfra.StartPostgres)
}
func (inf *Infra) MySQL(t *testing.T) string  { return inf.get(t, "mysql", testinfra.StartMySQL) }
func (inf *Infra) Mongo(t *testing.T) string  { return inf.get(t, "mongo", testinfra.StartMongo) }
func (inf *Infra) Redis(t *testing.T) string  { return inf.get(t, "redis", testinfra.StartRedis) }
func (inf *Infra) Rabbit(t *testing.T) string { return inf.get(t, "rabbit", testinfra.StartRabbit) }
func (inf *Infra) Kafka(t *testing.T) string  { return inf.get(t, "kafka", testinfra.StartKafka) }
func (inf *Infra) NATS(t *testing.T) string   { return inf.get(t, "nats", testinfra.StartNATS) }
func (inf *Infra) OpenSearch(t *testing.T) string {
	return inf.get(t, "opensearch", testinfra.StartOpenSearch)
}
func (inf *Infra) ClickHouse(t *testing.T) string {
	return inf.get(t, "clickhouse", testinfra.StartClickHouse)
}
