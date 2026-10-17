//go:build e2e || chaos

package harness

import (
	"slices"
	"sync"
	"testing"

	"github.com/s4bb4t/infra-layer-practice/internal/app/port"
	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/contracttest"
)

type (
	StoreFactory  func(t *testing.T, inf *Infra) contracttest.Store
	BrokerFactory func(t *testing.T, inf *Infra, topic string) contracttest.Broker
	// CacheFactory wraps a repository with a cache (step 05).
	CacheFactory func(t *testing.T, inf *Infra, next port.OrderRepository) port.OrderRepository
	// Projection is a read model fed by a consumer group (steps 10, 11).
	Projection struct {
		Group   string
		Handler port.EventHandler
		// Fields are free-form: scenarios type-assert what they need,
		// e.g. Search.(port.OrderSearch).
		Search    port.OrderSearch
		Analytics port.SalesAnalytics
	}
	ProjectionFactory func(t *testing.T, inf *Infra) Projection
)

var (
	mu          sync.Mutex
	stores      = map[string]StoreFactory{}
	brokers     = map[string]BrokerFactory{}
	caches      = map[string]CacheFactory{}
	projections = map[string]ProjectionFactory{}
)

// Register* are called from init() in factories files. Your own adapters
// (NATS, Redis cache, OpenSearch, ClickHouse) are registered the same way in
// test/e2e/factories_test.go.
func RegisterStore(name string, f StoreFactory)   { mu.Lock(); stores[name] = f; mu.Unlock() }
func RegisterBroker(name string, f BrokerFactory) { mu.Lock(); brokers[name] = f; mu.Unlock() }
func RegisterCache(name string, f CacheFactory)   { mu.Lock(); caches[name] = f; mu.Unlock() }
func RegisterProjection(name string, f ProjectionFactory) {
	mu.Lock()
	projections[name] = f
	mu.Unlock()
}

func names[V any](m map[string]V) []string {
	mu.Lock()
	defer mu.Unlock()
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

func StoreNames() []string      { return names(stores) }
func BrokerNames() []string     { return names(brokers) }
func CacheNames() []string      { return names(caches) }
func ProjectionNames() []string { return names(projections) }
