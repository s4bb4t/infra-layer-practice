// Package usecase orchestrates the domain through ports.
//
// Note what is NOT imported here: no SQL, no broker client, no cache client.
// If you ever need to import something from internal/infrastructure in this
// package, the design is broken.
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/s4bb4t/infra-layer-practice/internal/app/domain"
	"github.com/s4bb4t/infra-layer-practice/internal/app/port"
)

// Deps groups the ports the write use cases need.
type Deps struct {
	Tx     port.TxManager
	Orders port.OrderRepository
	Outbox port.Outbox
	Clock  port.Clock
	IDs    port.IDGenerator
}

type PlaceOrderInput struct {
	CustomerID domain.CustomerID
	Items      []domain.Item
}

// PlaceOrder creates an order and records OrderPlaced in the outbox atomically.
//
// It does not publish to a broker, update a cache, a search index or analytics:
// the outbox relay publishes the event, and consumers build read models.
type PlaceOrder struct{ d Deps }

func NewPlaceOrder(d Deps) *PlaceOrder { return &PlaceOrder{d: d} }

func (uc *PlaceOrder) Execute(ctx context.Context, in PlaceOrderInput) (domain.Order, error) {
	now := uc.d.Clock.Now()
	o, err := domain.NewOrder(domain.OrderID(uc.d.IDs.NewID()), in.CustomerID, in.Items, now)
	if err != nil {
		return domain.Order{}, err
	}

	err = uc.d.Tx.Do(ctx, func(ctx context.Context) error {
		if err := uc.d.Orders.Create(ctx, o); err != nil {
			return fmt.Errorf("create order: %w", err)
		}
		if err := uc.d.Outbox.Add(ctx, domain.NewOrderPlaced(uc.d.IDs.NewID(), o, now)); err != nil {
			return fmt.Errorf("add to outbox: %w", err)
		}
		return nil
	})
	if err != nil {
		return domain.Order{}, err
	}
	return o, nil
}

// PayOrder and CancelOrder share the same shape: load, transition, save, record event.
type PayOrder struct{ d Deps }

func NewPayOrder(d Deps) *PayOrder { return &PayOrder{d: d} }

func (uc *PayOrder) Execute(ctx context.Context, id domain.OrderID) (domain.Order, error) {
	return transition(ctx, uc.d, id, (*domain.Order).Pay, domain.NewOrderPaid)
}

type CancelOrder struct{ d Deps }

func NewCancelOrder(d Deps) *CancelOrder { return &CancelOrder{d: d} }

func (uc *CancelOrder) Execute(ctx context.Context, id domain.OrderID) (domain.Order, error) {
	return transition(ctx, uc.d, id, (*domain.Order).Cancel, domain.NewOrderCancelled)
}

func transition(
	ctx context.Context,
	d Deps,
	id domain.OrderID,
	apply func(*domain.Order) error,
	event func(string, domain.Order, time.Time) domain.Event,
) (domain.Order, error) {
	var out domain.Order
	err := d.Tx.Do(ctx, func(ctx context.Context) error {
		o, err := d.Orders.Get(ctx, id)
		if err != nil {
			return fmt.Errorf("get order: %w", err)
		}
		if err := apply(&o); err != nil {
			return err
		}
		if err := d.Orders.Update(ctx, o); err != nil {
			return fmt.Errorf("update order: %w", err)
		}
		o.Version++
		if err := d.Outbox.Add(ctx, event(d.IDs.NewID(), o, d.Clock.Now())); err != nil {
			return fmt.Errorf("add to outbox: %w", err)
		}
		out = o
		return nil
	})
	if err != nil {
		return domain.Order{}, err
	}
	return out, nil
}

// GetOrder is a plain read. Whether it hits a cache is decided by wiring in
// main.go (a caching decorator around OrderRepository), not here.
type GetOrder struct{ orders port.OrderRepository }

func NewGetOrder(orders port.OrderRepository) *GetOrder { return &GetOrder{orders: orders} }

func (uc *GetOrder) Execute(ctx context.Context, id domain.OrderID) (domain.Order, error) {
	return uc.orders.Get(ctx, id)
}
