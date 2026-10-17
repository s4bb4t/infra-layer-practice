//go:build chaos

// Package chaos breaks the infrastructure on purpose and checks that the
// service keeps its promises: no lost orders, no lost events, no half-done
// changes. Run with `make chaos`.
package chaos_test

import (
	"testing"
	"time"

	"github.com/s4bb4t/infra-layer-practice/internal/app/domain"
	"github.com/s4bb4t/infra-layer-practice/internal/app/usecase"
	"github.com/s4bb4t/infra-layer-practice/test/e2e/harness"
)

func TestMain(m *testing.M) { harness.Main(m) }

var stack = harness.Stack{Store: "postgres-toxic", Broker: "rabbit-toxic"}

func place(t *testing.T, app *harness.App) domain.Order {
	t.Helper()
	o, err := app.Place.Execute(app.Ctx, usecase.PlaceOrderInput{
		CustomerID: "alice",
		Items:      []domain.Item{{SKU: "book", Qty: 1, PriceCents: 1500}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return o
}

// Example: the broker is down for a while. Orders are still accepted
// (the outbox is in Postgres), and every event arrives after recovery.
func TestBrokerOutageLosesNothing(t *testing.T) {
	env := harness.StartToxic(t)
	app := harness.Build(t, stack)
	rec := app.Record(t)

	if err := env.Rabbit.Disable(); err != nil {
		t.Fatal(err)
	}
	var ids []domain.OrderID
	for range 5 {
		ids = append(ids, place(t, app).ID)
	}
	time.Sleep(2 * time.Second) // the relay keeps failing meanwhile
	if err := env.Rabbit.Enable(); err != nil {
		t.Fatal(err)
	}

	harness.Eventually(t, 60*time.Second, func() bool {
		for _, id := range ids {
			if len(rec.Of(id)) == 0 {
				return false
			}
		}
		return true
	}, "all OrderPlaced events delivered after the broker came back")
}
