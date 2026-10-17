package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/s4bb4t/infra-layer-practice/internal/app/domain"
	"github.com/s4bb4t/infra-layer-practice/internal/app/port"
	"github.com/s4bb4t/infra-layer-practice/internal/todo"
)

var (
	_ port.Outbox       = (*Outbox)(nil)
	_ port.OutboxReader = (*Outbox)(nil)
)

type Outbox struct{ db *mongo.Database }

func NewOutbox(db *mongo.Database) *Outbox { return &Outbox{db: db} }

// Add — TODO(step-04). Контракт требует: Fetch отдаёт события в порядке
// добавления, даже между разными транзакциями. В Mongo нет AUTO_INCREMENT,
// а occurred_at у событий может совпадать. Чем упорядочить?
func (o *Outbox) Add(ctx context.Context, events ...domain.Event) error {
	return todo.Step("04")
}

func (o *Outbox) Fetch(ctx context.Context, limit int) ([]domain.Event, error) {
	return nil, todo.Step("04")
}

func (o *Outbox) MarkPublished(ctx context.Context, ids ...string) error {
	return todo.Step("04")
}
