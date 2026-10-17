// Package mongo implements the storage ports on MongoDB — шаг 04.
//
// Контракт-тест (mongo_test.go) и контейнер (testinfra.StartMongo) уже даны.
// Сигнатуры ниже — тоже. Всё остальное — твоё.
//
// Multi-document transactions need a replica set (a single-node one is fine,
// testinfra starts exactly that).
package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/s4bb4t/infra-layer-practice/internal/app/port"
	"github.com/s4bb4t/infra-layer-practice/internal/todo"
)

// EnsureIndexes is Mongo's "migration": collections appear on first write,
// indexes do not.
//
// TODO(step-04): какие индексы нужны под ListByCustomer и под Fetch из outbox?
// Подсказка: partial index.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	return nil
}

var _ port.TxManager = (*TxManager)(nil)

type TxManager struct{ client *mongo.Client }

func NewTxManager(client *mongo.Client) *TxManager { return &TxManager{client: client} }

// Do — TODO(step-04): сессия + транзакция. Прочитай `go doc mongo.Session.WithTransaction`
// целиком: что он делает с fn при транзиентных ошибках и что из этого следует для fn?
func (m *TxManager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	return todo.Step("04")
}
