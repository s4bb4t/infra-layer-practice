//go:build e2e || chaos

package harness

import (
	"context"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/s4bb4t/infra-layer-practice/internal/app/domain"
	"github.com/s4bb4t/infra-layer-practice/internal/todo"
)

// Scenario is one end-to-end story, run on every stack of the matrix.
type Scenario struct {
	Name string
	// Stack adjusts the matrix stack for this scenario (add a cache or
	// projections). nil means "as is".
	Stack func(Stack) Stack
	Run   func(t *testing.T, app *App)
}

// RunMatrix runs every scenario on every registered store x broker.
//
// Narrow it down while working:
//
//	E2E_STORES=postgres,mongo E2E_BROKERS=kafka make e2e
func RunMatrix(t *testing.T, scenarios []Scenario) {
	for _, store := range filter(StoreNames(), "E2E_STORES") {
		for _, broker := range filter(BrokerNames(), "E2E_BROKERS") {
			base := Stack{Store: store, Broker: broker}
			t.Run(base.String(), func(t *testing.T) {
				for _, sc := range scenarios {
					t.Run(sc.Name, func(t *testing.T) {
						s := base
						if sc.Stack != nil {
							s = sc.Stack(s)
						}
						app := Build(t, s)
						skipIfTODO(t, app)
						sc.Run(t, app)
					})
				}
			})
		}
	}
}

func filter(all []string, env string) []string {
	v := os.Getenv(env)
	if v == "" {
		return all
	}
	want := strings.Split(v, ",")
	return slices.DeleteFunc(slices.Clone(all), func(s string) bool { return !slices.Contains(want, s) })
}

// skipIfTODO probes the adapters: a stack with an unfinished step is skipped,
// not failed.
func skipIfTODO(t *testing.T, app *App) {
	t.Helper()
	ctx := context.Background()
	probes := []error{
		app.Store.Tx.Do(ctx, func(context.Context) error { return nil }),
		func() error { _, err := app.Store.Orders.Get(ctx, "probe"); return err }(),
		func() error { _, err := app.Store.Reader.Fetch(ctx, 1); return err }(),
		app.Broker.Sub.Subscribe(app.Ctx, "probe-"+suffix(), func(context.Context, domain.Event) error { return nil }),
	}
	for _, err := range probes {
		if errors.Is(err, todo.ErrNotImplemented) {
			t.Skip(err.Error())
		}
	}
}
