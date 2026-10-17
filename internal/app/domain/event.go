package domain

import (
	"encoding/json"
	"time"
)

type EventType string

const (
	EventOrderPlaced    EventType = "order.placed"
	EventOrderPaid      EventType = "order.paid"
	EventOrderCancelled EventType = "order.cancelled"
)

// Event is the envelope that travels through the outbox and the broker.
//
// ID is unique per event: consumers use it for deduplication, because every
// broker in this project delivers at least once.
// Version is the aggregate version the event was produced at: consumers can
// use it to drop out-of-order updates.
type Event struct {
	ID          string          `json:"id"`
	Type        EventType       `json:"type"`
	OccurredAt  time.Time       `json:"occurred_at"`
	AggregateID OrderID         `json:"aggregate_id"`
	Version     int             `json:"version"`
	Payload     json.RawMessage `json:"payload"`
}

// OrderPlacedPayload carries a full snapshot so that read models
// (search, analytics) can be built without calling back into the service.
type OrderPlacedPayload struct {
	Order Order `json:"order"`
}

// OrderStatusPayload is used by paid/cancelled events.
type OrderStatusPayload struct {
	Status Status `json:"status"`
}

func NewOrderPlaced(eventID string, o Order, now time.Time) Event {
	return newEvent(eventID, EventOrderPlaced, o, now, OrderPlacedPayload{Order: o})
}

func NewOrderPaid(eventID string, o Order, now time.Time) Event {
	return newEvent(eventID, EventOrderPaid, o, now, OrderStatusPayload{Status: o.Status})
}

func NewOrderCancelled(eventID string, o Order, now time.Time) Event {
	return newEvent(eventID, EventOrderCancelled, o, now, OrderStatusPayload{Status: o.Status})
}

func newEvent(id string, typ EventType, o Order, now time.Time, payload any) Event {
	raw, err := json.Marshal(payload)
	if err != nil {
		// Payload types above are plain structs: Marshal cannot fail on them.
		panic(err)
	}
	return Event{
		ID:          id,
		Type:        typ,
		OccurredAt:  now.UTC(),
		AggregateID: o.ID,
		Version:     o.Version,
		Payload:     raw,
	}
}
