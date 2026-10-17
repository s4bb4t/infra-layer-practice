package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/s4bb4t/infra-layer-practice/internal/app/domain"
	"github.com/s4bb4t/infra-layer-practice/internal/app/port"
	"github.com/s4bb4t/infra-layer-practice/internal/todo"
)

var _ port.OrderRepository = (*Orders)(nil)

// TODO(step-04): заведи отдельную структуру документа с bson-тегами.
// Почему не повесить bson-теги прямо на domain.Order?

type Orders struct{ coll *mongo.Collection }

func NewOrders(db *mongo.Database) *Orders { return &Orders{coll: db.Collection("orders")} }

func (r *Orders) Create(ctx context.Context, o domain.Order) error {
	return todo.Step("04")
}

func (r *Orders) Get(ctx context.Context, id domain.OrderID) (domain.Order, error) {
	return domain.Order{}, todo.Step("04")
}

// Update — optimistic locking в Mongo: фильтр по _id И version, $inc для версии.
// Как по результату UpdateOne отличить ErrNotFound от ErrConflict?
func (r *Orders) Update(ctx context.Context, o domain.Order) error {
	return todo.Step("04")
}

func (r *Orders) ListByCustomer(ctx context.Context, customer domain.CustomerID, limit int) ([]domain.Order, error) {
	return nil, todo.Step("04")
}
