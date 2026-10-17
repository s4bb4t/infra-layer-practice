package outbox_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/s4bb4t/infra-layer-practice/internal/app/domain"
	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/memory"
	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/outbox"
)

var discard = slog.New(slog.NewTextHandler(io.Discard, nil))

type recordingPub struct {
	got  []domain.Event
	fail error
}

func (p *recordingPub) Publish(_ context.Context, events ...domain.Event) error {
	if p.fail != nil {
		return p.fail
	}
	p.got = append(p.got, events...)
	return nil
}

func addEvents(t *testing.T, s *memory.Store, n int) {
	t.Helper()
	o, _ := domain.NewOrder("o1", "c1", []domain.Item{{SKU: "a", Qty: 1}}, time.Now())
	for i := range n {
		e := domain.NewOrderPlaced(string(rune('a'+i)), o, time.Now())
		if err := s.Add(context.Background(), e); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRelayPublishesAndMarks(t *testing.T) {
	ctx := context.Background()
	s := memory.NewStore()
	addEvents(t, s, 3)
	pub := &recordingPub{}
	r := outbox.NewRelay(s, s, pub, discard)

	n, err := r.RelayOnce(ctx)
	if err != nil || n != 3 || len(pub.got) != 3 {
		t.Fatalf("n=%d err=%v published=%d", n, err, len(pub.got))
	}
	left, _ := s.Fetch(ctx, 10)
	if len(left) != 0 {
		t.Fatalf("%d events left unpublished", len(left))
	}
	if n, _ := r.RelayOnce(ctx); n != 0 {
		t.Fatalf("second run relayed %d", n)
	}
}

func TestRelayKeepsEventsWhenBrokerFails(t *testing.T) {
	ctx := context.Background()
	s := memory.NewStore()
	addEvents(t, s, 2)
	pub := &recordingPub{fail: errors.New("broker down")}
	r := outbox.NewRelay(s, s, pub, discard)

	if _, err := r.RelayOnce(ctx); err == nil {
		t.Fatal("want error")
	}
	left, _ := s.Fetch(ctx, 10)
	if len(left) != 2 {
		t.Fatalf("%d events left, want 2", len(left))
	}

	pub.fail = nil
	if n, err := r.RelayOnce(ctx); err != nil || n != 2 {
		t.Fatalf("retry: n=%d err=%v", n, err)
	}
}
