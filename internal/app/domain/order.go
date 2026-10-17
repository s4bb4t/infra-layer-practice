// Package domain contains the business core of the orders service.
//
// It imports nothing from the project and nothing infrastructural from the
// outside world: only the standard library. Every other package depends on
// domain, never the other way around.
package domain

import (
	"fmt"
	"time"
)

type (
	OrderID    string
	CustomerID string
)

type Status string

const (
	StatusCreated   Status = "created"
	StatusPaid      Status = "paid"
	StatusCancelled Status = "cancelled"
)

// Item is a single order line. Money is always stored in int64 cents:
// floats cannot represent 0.1 exactly and lose money on rounding.
type Item struct {
	SKU        string `json:"sku"`
	Qty        int    `json:"qty"`
	PriceCents int64  `json:"price_cents"`
}

// Order is the aggregate root.
//
// Version is used for optimistic locking: a storage adapter must refuse an
// update if the stored version differs from Order.Version and must bump the
// stored version by one on success.
type Order struct {
	ID         OrderID    `json:"id"`
	CustomerID CustomerID `json:"customer_id"`
	Items      []Item     `json:"items"`
	TotalCents int64      `json:"total_cents"`
	Status     Status     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	Version    int        `json:"version"`
}

// NewOrder validates input and builds an order in StatusCreated with Version 1.
func NewOrder(id OrderID, customer CustomerID, items []Item, now time.Time) (Order, error) {
	if id == "" {
		return Order{}, fmt.Errorf("%w: empty id", ErrInvalidOrder)
	}
	if customer == "" {
		return Order{}, fmt.Errorf("%w: empty customer", ErrInvalidOrder)
	}
	if len(items) == 0 {
		return Order{}, fmt.Errorf("%w: no items", ErrInvalidOrder)
	}

	var total int64
	for i, it := range items {
		if it.SKU == "" {
			return Order{}, fmt.Errorf("%w: item %d has empty sku", ErrInvalidOrder, i)
		}
		if it.Qty <= 0 {
			return Order{}, fmt.Errorf("%w: item %d has non-positive qty", ErrInvalidOrder, i)
		}
		if it.PriceCents < 0 {
			return Order{}, fmt.Errorf("%w: item %d has negative price", ErrInvalidOrder, i)
		}
		total += int64(it.Qty) * it.PriceCents
	}

	return Order{
		ID:         id,
		CustomerID: customer,
		Items:      append([]Item(nil), items...),
		TotalCents: total,
		Status:     StatusCreated,
		CreatedAt:  now.UTC(),
		Version:    1,
	}, nil
}

// Pay moves a created order to paid.
func (o *Order) Pay() error {
	if o.Status != StatusCreated {
		return fmt.Errorf("%w: pay from %s", ErrInvalidTransition, o.Status)
	}
	o.Status = StatusPaid
	return nil
}

// Cancel moves a created or paid order to cancelled.
func (o *Order) Cancel() error {
	if o.Status == StatusCancelled {
		return fmt.Errorf("%w: cancel from %s", ErrInvalidTransition, o.Status)
	}
	o.Status = StatusCancelled
	return nil
}
