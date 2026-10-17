package http_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/s4bb4t/infra-layer-practice/internal/app/domain"
	"github.com/s4bb4t/infra-layer-practice/internal/app/usecase"
	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/memory"
	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/system"
	transport "github.com/s4bb4t/infra-layer-practice/internal/presentation/http"
)

func TestOrdersAPI(t *testing.T) {
	s := memory.NewStore()
	deps := usecase.Deps{Tx: s, Orders: s, Outbox: s, Clock: system.Clock{}, IDs: system.UUIDv7{}}
	h := transport.NewHandler(transport.UseCases{
		Place:  usecase.NewPlaceOrder(deps),
		Pay:    usecase.NewPayOrder(deps),
		Cancel: usecase.NewCancelOrder(deps),
		Get:    usecase.NewGetOrder(s),
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	srv := httptest.NewServer(h)
	defer srv.Close()

	do := func(method, path, body string) (int, domain.Order) {
		t.Helper()
		req, _ := http.NewRequestWithContext(t.Context(), method, srv.URL+path, strings.NewReader(body))
		resp, err := srv.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = resp.Body.Close() }()
		var o domain.Order
		_ = json.NewDecoder(resp.Body).Decode(&o)
		return resp.StatusCode, o
	}

	code, o := do("POST", "/orders", `{"customer_id":"c1","items":[{"sku":"a","qty":2,"price_cents":100}]}`)
	if code != http.StatusCreated || o.TotalCents != 200 {
		t.Fatalf("place: %d %+v", code, o)
	}
	if code, _ := do("POST", "/orders", `{"customer_id":"c1"}`); code != http.StatusBadRequest {
		t.Fatalf("invalid place: %d", code)
	}
	if code, got := do("GET", "/orders/"+string(o.ID), ""); code != http.StatusOK || got.ID != o.ID {
		t.Fatalf("get: %d %+v", code, got)
	}
	if code, _ := do("GET", "/orders/missing", ""); code != http.StatusNotFound {
		t.Fatalf("get missing: %d", code)
	}
	if code, got := do("POST", "/orders/"+string(o.ID)+"/pay", ""); code != http.StatusOK || got.Status != domain.StatusPaid {
		t.Fatalf("pay: %d %+v", code, got)
	}
	if code, _ := do("POST", "/orders/"+string(o.ID)+"/pay", ""); code != http.StatusConflict {
		t.Fatalf("double pay: %d", code)
	}
}
