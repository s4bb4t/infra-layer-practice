package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/s4bb4t/infra-layer-practice/internal/app/domain"
	"github.com/s4bb4t/infra-layer-practice/internal/app/port"
	"github.com/s4bb4t/infra-layer-practice/internal/todo"
)

var _ port.OrderRepository = (*Orders)(nil)

type Orders struct{ db *sql.DB }

func NewOrders(db *sql.DB) *Orders { return &Orders{db: db} }

const orderColumns = `id, customer_id, items, total_cents, status, created_at, version`

func (r *Orders) Create(ctx context.Context, o domain.Order) error {
	items, err := json.Marshal(o.Items)
	if err != nil {
		return err
	}
	_, err = conn(ctx, r.db).ExecContext(ctx,
		`INSERT INTO orders (`+orderColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		o.ID, o.CustomerID, items, o.TotalCents, o.Status, o.CreatedAt, o.Version)
	if isDuplicate(err) {
		return domain.ErrAlreadyExists
	}
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}
	return nil
}

func (r *Orders) Get(ctx context.Context, id domain.OrderID) (domain.Order, error) {
	row := conn(ctx, r.db).QueryRowContext(ctx, `SELECT `+orderColumns+` FROM orders WHERE id = ?`, id)
	o, err := scanOrder(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Order{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Order{}, fmt.Errorf("select order: %w", err)
	}
	return o, nil
}

// Update — шаг 03.
//
// TODO(step-03): optimistic locking, как в postgres.Orders.Update:
//   - UPDATE ... SET ..., version = version + 1 WHERE id = ? AND version = ?;
//   - 0 затронутых строк → отличи ErrNotFound от ErrConflict;
//   - осторожно: RowsAffected в MySQL по умолчанию считает ИЗМЕНЁННЫЕ строки,
//     а не НАЙДЕННЫЕ (см. clientFoundRows в DSN). Почему здесь это не ломает логику?
func (r *Orders) Update(ctx context.Context, o domain.Order) error {
	return todo.Step("03")
}

func (r *Orders) ListByCustomer(ctx context.Context, customer domain.CustomerID, limit int) ([]domain.Order, error) {
	rows, err := conn(ctx, r.db).QueryContext(ctx,
		`SELECT `+orderColumns+` FROM orders WHERE customer_id = ? ORDER BY created_at DESC LIMIT ?`,
		customer, limit)
	if err != nil {
		return nil, fmt.Errorf("select orders: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []domain.Order
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

type scanner interface{ Scan(dest ...any) error }

func scanOrder(row scanner) (domain.Order, error) {
	var (
		o     domain.Order
		items []byte
	)
	if err := row.Scan(&o.ID, &o.CustomerID, &items, &o.TotalCents, &o.Status, &o.CreatedAt, &o.Version); err != nil {
		return domain.Order{}, err
	}
	if err := json.Unmarshal(items, &o.Items); err != nil {
		return domain.Order{}, fmt.Errorf("decode items: %w", err)
	}
	o.CreatedAt = o.CreatedAt.UTC()
	return o, nil
}
