// Package outbox moves events from the transactional outbox to a broker.
//
// It depends only on ports, so the same relay works for any storage
// (Postgres, MySQL, Mongo, memory) and any broker.
package outbox

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/s4bb4t/infra-layer-practice/internal/app/domain"
	"github.com/s4bb4t/infra-layer-practice/internal/app/port"
)

type Relay struct {
	tx       port.TxManager
	reader   port.OutboxReader
	pub      port.EventPublisher
	batch    int
	interval time.Duration
	log      *slog.Logger
}

func NewRelay(tx port.TxManager, reader port.OutboxReader, pub port.EventPublisher, log *slog.Logger) *Relay {
	return &Relay{tx: tx, reader: reader, pub: pub, batch: 100, interval: 200 * time.Millisecond, log: log}
}

// Run polls the outbox until ctx is cancelled.
func (r *Relay) Run(ctx context.Context) error {
	for {
		n, err := r.RelayOnce(ctx)
		if err != nil {
			r.log.WarnContext(ctx, "outbox relay failed", "err", err)
		}
		// A full batch means there is probably more: do not sleep.
		if err == nil && n == r.batch {
			continue
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(r.interval):
		}
	}
}

// RelayOnce publishes one batch and returns how many events were relayed.
//
// Fetch, Publish and MarkPublished run in one transaction: on storages that
// lock fetched rows (FOR UPDATE SKIP LOCKED) several relays do not publish the
// same batch. If MarkPublished or COMMIT fails after Publish succeeded, the
// batch is published again on the next run: delivery is at-least-once.
func (r *Relay) RelayOnce(ctx context.Context) (int, error) {
	var n int
	err := r.tx.Do(ctx, func(ctx context.Context) error {
		events, err := r.reader.Fetch(ctx, r.batch)
		if err != nil {
			return fmt.Errorf("fetch: %w", err)
		}
		if len(events) == 0 {
			return nil
		}
		if err := r.pub.Publish(ctx, events...); err != nil {
			return fmt.Errorf("publish: %w", err)
		}
		if err := r.reader.MarkPublished(ctx, ids(events)...); err != nil {
			return fmt.Errorf("mark published: %w", err)
		}
		n = len(events)
		return nil
	})
	return n, err
}

func ids(events []domain.Event) []string {
	out := make([]string, len(events))
	for i, e := range events {
		out[i] = e.ID
	}
	return out
}
