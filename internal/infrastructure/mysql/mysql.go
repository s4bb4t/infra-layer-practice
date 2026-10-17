// Package mysql implements the storage ports on MySQL 8 with database/sql.
//
// Compare it with package postgres line by line: the ports and the contract
// are the same, the dialect details are not.
package mysql

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"

	mysqldrv "github.com/go-sql-driver/mysql"
	"github.com/pressly/goose/v3"

	"github.com/s4bb4t/infra-layer-practice/internal/app/port"
	"github.com/s4bb4t/infra-layer-practice/internal/todo"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Migrate(ctx context.Context, db *sql.DB) error {
	sub, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return err
	}
	p, err := goose.NewProvider(goose.DialectMySQL, db, sub)
	if err != nil {
		return fmt.Errorf("goose provider: %w", err)
	}
	if _, err := p.Up(ctx); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

// querier is what both *sql.DB and *sql.Tx can do.
type querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type txKey struct{}

func conn(ctx context.Context, db *sql.DB) querier {
	if tx, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
		return tx
	}
	return db
}

var _ port.TxManager = (*TxManager)(nil)

type TxManager struct{ db *sql.DB }

func NewTxManager(db *sql.DB) *TxManager { return &TxManager{db: db} }

// Do — шаг 03.
//
// TODO(step-03): посмотри postgres.TxManager.Do и повтори на database/sql:
//   - вложенный вызов присоединяется к внешней транзакции (tx уже лежит в ctx);
//   - ошибка fn → Rollback, успех → Commit;
//   - транзакция кладётся в ctx под txKey{}, conn() её оттуда достанет.
//
// Подумай: какой уровень изоляции у InnoDB по умолчанию и чем он отличается
// от READ COMMITTED в Postgres? Нужно ли его менять для этого кода?
func (m *TxManager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	return todo.Step("03")
}

func isDuplicate(err error) bool {
	var myErr *mysqldrv.MySQLError
	return errors.As(err, &myErr) && myErr.Number == 1062
}
