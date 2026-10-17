package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/s4bb4t/infra-layer-practice/internal/app/domain"
	"github.com/s4bb4t/infra-layer-practice/internal/app/port"
)

var (
	_ port.Outbox       = (*Outbox)(nil)
	_ port.OutboxReader = (*Outbox)(nil)
)

type Outbox struct{ pool *pgxpool.Pool }

func NewOutbox(pool *pgxpool.Pool) *Outbox { return &Outbox{pool: pool} }

func (o *Outbox) Add(ctx context.Context, events ...domain.Event) error {
	q := conn(ctx, o.pool)
	for _, e := range events {
		_, err := q.Exec(ctx,
			`INSERT INTO outbox (id, type, occurred_at, aggregate_id, version, payload)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			e.ID, e.Type, e.OccurredAt, e.AggregateID, e.Version, []byte(e.Payload))
		if err != nil {
			return fmt.Errorf("insert outbox: %w", err)
		}
	}
	return nil
}

// Fetch locks the rows it returns when called inside a transaction.
// SKIP LOCKED lets several relay instances work in parallel without
// handing out the same event twice.
func (o *Outbox) Fetch(ctx context.Context, limit int) ([]domain.Event, error) {
	rows, err := conn(ctx, o.pool).Query(ctx,
		`SELECT id, type, occurred_at, aggregate_id, version, payload
		 FROM outbox WHERE published_at IS NULL
		 ORDER BY seq LIMIT $1
		 FOR UPDATE SKIP LOCKED`, limit)
	if err != nil {
		return nil, fmt.Errorf("select outbox: %w", err)
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Event, error) {
		var (
			e       domain.Event
			payload []byte
		)
		err := row.Scan(&e.ID, &e.Type, &e.OccurredAt, &e.AggregateID, &e.Version, &payload)
		e.Payload = payload
		e.OccurredAt = e.OccurredAt.UTC()
		return e, err
	})
}

func (o *Outbox) MarkPublished(ctx context.Context, ids ...string) error {
	_, err := conn(ctx, o.pool).Exec(ctx, `UPDATE outbox SET published_at = now() WHERE id = ANY($1)`, ids)
	if err != nil {
		return fmt.Errorf("mark published: %w", err)
	}
	return nil
}
