//go:build e2e

package e2e_test

import (
	"testing"
	"time"

	"github.com/s4bb4t/infra-layer-practice/internal/app/domain"
	"github.com/s4bb4t/infra-layer-practice/internal/app/usecase"
	"github.com/s4bb4t/infra-layer-practice/test/e2e/harness"
)

func TestMain(m *testing.M) { harness.Main(m) }

// scenarios is the list RunMatrix runs on every stack. The first one is an
// example; add yours in scenarios_test.go (append in init()).
var scenarios = []harness.Scenario{
	{
		Name: "placed order reaches the broker",
		Run: func(t *testing.T, app *harness.App) {
			rec := app.Record(t)

			o, err := app.Place.Execute(app.Ctx, usecase.PlaceOrderInput{
				CustomerID: "alice",
				Items:      []domain.Item{{SKU: "book", Qty: 1, PriceCents: 1500}},
			})
			if err != nil {
				t.Fatal(err)
			}

			harness.Eventually(t, 30*time.Second, func() bool {
				return len(rec.Of(o.ID)) == 1
			}, "OrderPlaced delivered")

			e := rec.Of(o.ID)[0]
			if e.Type != domain.EventOrderPlaced || e.Version != 1 {
				t.Fatalf("got %+v", e)
			}
		},
	},
}

func TestMatrix(t *testing.T) {
	harness.RunMatrix(t, scenarios)
}
