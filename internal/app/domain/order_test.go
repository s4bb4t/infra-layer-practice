package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/s4bb4t/infra-layer-practice/internal/app/domain"
)

var now = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

func TestNewOrder(t *testing.T) {
	items := []domain.Item{{SKU: "a", Qty: 2, PriceCents: 150}, {SKU: "b", Qty: 1, PriceCents: 1000}}

	o, err := domain.NewOrder("o1", "c1", items, now)
	if err != nil {
		t.Fatal(err)
	}
	if o.TotalCents != 1300 {
		t.Errorf("total = %d, want 1300", o.TotalCents)
	}
	if o.Status != domain.StatusCreated || o.Version != 1 {
		t.Errorf("got status %s version %d", o.Status, o.Version)
	}

	items[0].Qty = 100
	if o.Items[0].Qty != 2 {
		t.Error("order must not share the items slice with the caller")
	}
}

func TestNewOrderValidation(t *testing.T) {
	cases := map[string]struct {
		id       domain.OrderID
		customer domain.CustomerID
		items    []domain.Item
	}{
		"empty id":       {"", "c", []domain.Item{{SKU: "a", Qty: 1}}},
		"empty customer": {"o", "", []domain.Item{{SKU: "a", Qty: 1}}},
		"no items":       {"o", "c", nil},
		"empty sku":      {"o", "c", []domain.Item{{Qty: 1}}},
		"zero qty":       {"o", "c", []domain.Item{{SKU: "a"}}},
		"negative price": {"o", "c", []domain.Item{{SKU: "a", Qty: 1, PriceCents: -1}}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := domain.NewOrder(tc.id, tc.customer, tc.items, now)
			if !errors.Is(err, domain.ErrInvalidOrder) {
				t.Fatalf("err = %v, want ErrInvalidOrder", err)
			}
		})
	}
}

func TestTransitions(t *testing.T) {
	newOrder := func() domain.Order {
		o, err := domain.NewOrder("o", "c", []domain.Item{{SKU: "a", Qty: 1}}, now)
		if err != nil {
			t.Fatal(err)
		}
		return o
	}

	o := newOrder()
	if err := o.Pay(); err != nil {
		t.Fatal(err)
	}
	if err := o.Pay(); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("double pay: err = %v", err)
	}
	if err := o.Cancel(); err != nil {
		t.Fatalf("cancel paid: %v", err)
	}
	if err := o.Cancel(); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("double cancel: err = %v", err)
	}

	o = newOrder()
	if err := o.Cancel(); err != nil {
		t.Fatal(err)
	}
	if err := o.Pay(); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("pay cancelled: err = %v", err)
	}
}
