package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/s4bb4t/infra-layer-practice/internal/app/domain"
	"github.com/s4bb4t/infra-layer-practice/internal/app/port"
	"github.com/s4bb4t/infra-layer-practice/internal/todo"
)

var _ port.OrderRepository = (*Orders)(nil)

type Orders struct{ pool *pgxpool.Pool }

func NewOrders(pool *pgxpool.Pool) *Orders { return &Orders{pool: pool} }

const orderColumns = `id, customer_id, items, total_cents, status, created_at, version`

func (r *Orders) Create(ctx context.Context, o domain.Order) error {
	items, err := json.Marshal(o.Items)
	if err != nil {
		return err
	}
	_, err = conn(ctx, r.pool).Exec(ctx,
		`INSERT INTO orders (`+orderColumns+`) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		o.ID, o.CustomerID, items, o.TotalCents, o.Status, o.CreatedAt, o.Version)
	if isUniqueViolation(err) {
		return domain.ErrAlreadyExists
	}
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}
	return nil
}

func (r *Orders) Get(ctx context.Context, id domain.OrderID) (domain.Order, error) {
	row := conn(ctx, r.pool).QueryRow(ctx, `SELECT `+orderColumns+` FROM orders WHERE id = $1`, id)
	o, err := scanOrder(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Order{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Order{}, fmt.Errorf("select order: %w", err)
	}
	return o, nil
}

// Update relies on the WHERE version = $n check instead of SELECT ... FOR UPDATE:
// that is optimistic locking. Under READ COMMITTED a concurrent UPDATE of the
// same row waits for the first one and then re-checks the WHERE clause.
func (r *Orders) Update(ctx context.Context, o domain.Order) error {
	items, err := json.Marshal(o.Items)
	if err != nil {
		return err
	}
	q := conn(ctx, r.pool)
	tag, err := q.Exec(ctx,
		`UPDATE orders SET items = $3, total_cents = $4, status = $5, version = version + 1
		 WHERE id = $1 AND version = $2`,
		o.ID, o.Version, items, o.TotalCents, o.Status)
	if err != nil {
		return fmt.Errorf("update order: %w", err)
	}
	if tag.RowsAffected() == 1 {
		return nil
	}

	// Nothing updated: either the row is gone or the version moved on.
	var exists bool
	if err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM orders WHERE id = $1)`, o.ID).Scan(&exists); err != nil {
		return fmt.Errorf("check order: %w", err)
	}
	if !exists {
		return domain.ErrNotFound
	}
	return domain.ErrConflict
}

// ListByCustomer — шаг 02, расширение.
//
// TODO(step-02):
//   - напиши запрос: заказы клиента, новые первыми, не больше limit;
//   - добавь миграцию migrations/00002_*.sql с индексом под этот запрос;
//   - проверь план: EXPLAIN ANALYZE до и после индекса (результат — в ADR);
//   - не забудь rows.Err() и закрытие rows (линтер напомнит).
func (r *Orders) ListByCustomer(ctx context.Context, customer domain.CustomerID, limit int) ([]domain.Order, error) {
	return nil, todo.Step("02")
}

func scanOrder(row pgx.Row) (domain.Order, error) {
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
