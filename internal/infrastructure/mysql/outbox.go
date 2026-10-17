package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/s4bb4t/infra-layer-practice/internal/app/domain"
	"github.com/s4bb4t/infra-layer-practice/internal/app/port"
)

var (
	_ port.Outbox       = (*Outbox)(nil)
	_ port.OutboxReader = (*Outbox)(nil)
)

type Outbox struct{ db *sql.DB }

func NewOutbox(db *sql.DB) *Outbox { return &Outbox{db: db} }

func (o *Outbox) Add(ctx context.Context, events ...domain.Event) error {
	q := conn(ctx, o.db)
	for _, e := range events {
		_, err := q.ExecContext(ctx,
			`INSERT INTO outbox (id, type, occurred_at, aggregate_id, version, payload) VALUES (?, ?, ?, ?, ?, ?)`,
			e.ID, e.Type, e.OccurredAt, e.AggregateID, e.Version, []byte(e.Payload))
		if err != nil {
			return fmt.Errorf("insert outbox: %w", err)
		}
	}
	return nil
}

func (o *Outbox) Fetch(ctx context.Context, limit int) ([]domain.Event, error) {
	rows, err := conn(ctx, o.db).QueryContext(ctx,
		`SELECT id, type, occurred_at, aggregate_id, version, payload
		 FROM outbox WHERE published_at IS NULL
		 ORDER BY seq LIMIT ?
		 FOR UPDATE SKIP LOCKED`, limit)
	if err != nil {
		return nil, fmt.Errorf("select outbox: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []domain.Event
	for rows.Next() {
		var (
			e       domain.Event
			payload []byte
		)
		if err := rows.Scan(&e.ID, &e.Type, &e.OccurredAt, &e.AggregateID, &e.Version, &payload); err != nil {
			return nil, err
		}
		e.Payload = payload
		e.OccurredAt = e.OccurredAt.UTC()
		out = append(out, e)
	}
	return out, rows.Err()
}

func (o *Outbox) MarkPublished(ctx context.Context, ids ...string) error {
	if len(ids) == 0 {
		return nil
	}
	// database/sql has no array parameters: expand IN (?, ?, ...) by hand.
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	_, err := conn(ctx, o.db).ExecContext(ctx,
		`UPDATE outbox SET published_at = UTC_TIMESTAMP(6) WHERE id IN (`+placeholders+`)`, args...)
	if err != nil {
		return fmt.Errorf("mark published: %w", err)
	}
	return nil
}
