// Package contracttest holds behaviour specs for ports.
//
// A contract test is written once per port and run against every
// implementation of that port: memory, Postgres, MySQL, Mongo, ... If two
// adapters pass the same contract, use cases cannot tell them apart.
//
// This is a regular (non _test) package so that adapter test files in other
// packages can import it.
package contracttest

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/s4bb4t/infra-layer-practice/internal/app/domain"
)

// uniq returns a random suffix so that tests sharing one database do not collide.
func uniq(prefix string) string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return prefix + "-" + hex.EncodeToString(b[:])
}

// baseTime is truncated to milliseconds: the contract requires adapters to keep
// at least millisecond precision (Mongo and some drivers keep no more).
var baseTime = time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)

func newOrder(t *testing.T, customer domain.CustomerID, at time.Time) domain.Order {
	t.Helper()
	o, err := domain.NewOrder(domain.OrderID(uniq("order")), customer, []domain.Item{
		{SKU: "sku-1", Qty: 2, PriceCents: 1999},
		{SKU: "sku-2", Qty: 1, PriceCents: 500},
	}, at)
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func assertOrder(t *testing.T, got, want domain.Order) {
	t.Helper()
	if got.ID != want.ID || got.CustomerID != want.CustomerID || got.TotalCents != want.TotalCents ||
		got.Status != want.Status || got.Version != want.Version {
		t.Fatalf("order mismatch:\n got  %+v\n want %+v", got, want)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) {
		t.Fatalf("created_at: got %v, want %v", got.CreatedAt, want.CreatedAt)
	}
	if len(got.Items) != len(want.Items) {
		t.Fatalf("items: got %+v, want %+v", got.Items, want.Items)
	}
	for i := range want.Items {
		if got.Items[i] != want.Items[i] {
			t.Fatalf("item %d: got %+v, want %+v", i, got.Items[i], want.Items[i])
		}
	}
}

func newEvent(t *testing.T, o domain.Order) domain.Event {
	t.Helper()
	return domain.NewOrderPlaced(uniq("event"), o, baseTime)
}

// assertEvent compares payloads semantically: storages like Postgres JSONB
// do not preserve key order or whitespace.
func assertEvent(t *testing.T, got, want domain.Event) {
	t.Helper()
	if got.ID != want.ID || got.Type != want.Type || got.AggregateID != want.AggregateID || got.Version != want.Version {
		t.Fatalf("event mismatch:\n got  %+v\n want %+v", got, want)
	}
	if !got.OccurredAt.Equal(want.OccurredAt) {
		t.Fatalf("occurred_at: got %v, want %v", got.OccurredAt, want.OccurredAt)
	}
	if !jsonEqual(got.Payload, want.Payload) {
		t.Fatalf("payload: got %s, want %s", got.Payload, want.Payload)
	}
}

func jsonEqual(a, b []byte) bool {
	var x, y any
	if json.Unmarshal(a, &x) != nil || json.Unmarshal(b, &y) != nil {
		return false
	}
	ax, _ := json.Marshal(x)
	by, _ := json.Marshal(y)
	return bytes.Equal(ax, by)
}
