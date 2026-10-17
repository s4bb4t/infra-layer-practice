package memory_test

import (
	"testing"

	"github.com/s4bb4t/infra-layer-practice/internal/app/port"
	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/contracttest"
	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/memory"
)

func TestOrderRepositoryContract(t *testing.T) {
	contracttest.RunOrderRepository(t, func(t *testing.T) port.OrderRepository {
		return memory.NewStore()
	})
}

func TestTxOutboxContract(t *testing.T) {
	contracttest.RunTxOutbox(t, func(t *testing.T) contracttest.Store {
		s := memory.NewStore()
		return contracttest.Store{Tx: s, Orders: s, Outbox: s, Reader: s}
	})
}

func TestBrokerContract(t *testing.T) {
	contracttest.RunBroker(t, func(t *testing.T) contracttest.Broker {
		b := memory.NewBroker()
		return contracttest.Broker{Pub: b, Sub: b}
	})
}
