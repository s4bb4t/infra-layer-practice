// Package http is the transport layer: JSON over HTTP in, use cases out.
//
// It knows use cases and domain errors, and nothing about storage or brokers.
package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/s4bb4t/infra-layer-practice/internal/app/domain"
	"github.com/s4bb4t/infra-layer-practice/internal/app/usecase"
)

type UseCases struct {
	Place  *usecase.PlaceOrder
	Pay    *usecase.PayOrder
	Cancel *usecase.CancelOrder
	Get    *usecase.GetOrder
}

type Handler struct {
	uc  UseCases
	log *slog.Logger
	mux *http.ServeMux
}

func NewHandler(uc UseCases, log *slog.Logger) *Handler {
	h := &Handler{uc: uc, log: log, mux: http.NewServeMux()}
	h.mux.HandleFunc("POST /orders", h.place)
	h.mux.HandleFunc("GET /orders/{id}", h.get)
	h.mux.HandleFunc("POST /orders/{id}/pay", h.transition(uc.Pay.Execute))
	h.mux.HandleFunc("POST /orders/{id}/cancel", h.transition(uc.Cancel.Execute))
	return h
}

// Handle lets the composition root register extra routes (search, reports, metrics).
func (h *Handler) Handle(pattern string, handler http.Handler) { h.mux.Handle(pattern, handler) }

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) { h.mux.ServeHTTP(w, r) }

type placeRequest struct {
	CustomerID string        `json:"customer_id"`
	Items      []domain.Item `json:"items"`
}

func (h *Handler) place(w http.ResponseWriter, r *http.Request) {
	var req placeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, r, http.StatusBadRequest, err)
		return
	}
	o, err := h.uc.Place.Execute(r.Context(), usecase.PlaceOrderInput{
		CustomerID: domain.CustomerID(req.CustomerID),
		Items:      req.Items,
	})
	if err != nil {
		h.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, o)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	o, err := h.uc.Get.Execute(r.Context(), domain.OrderID(r.PathValue("id")))
	if err != nil {
		h.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func (h *Handler) transition(exec func(context.Context, domain.OrderID) (domain.Order, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		o, err := exec(r.Context(), domain.OrderID(r.PathValue("id")))
		if err != nil {
			h.writeDomainError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, o)
	}
}

func (h *Handler) writeDomainError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidOrder):
		h.writeError(w, r, http.StatusBadRequest, err)
	case errors.Is(err, domain.ErrNotFound):
		h.writeError(w, r, http.StatusNotFound, err)
	case errors.Is(err, domain.ErrInvalidTransition), errors.Is(err, domain.ErrConflict):
		h.writeError(w, r, http.StatusConflict, err)
	default:
		h.writeError(w, r, http.StatusInternalServerError, err)
	}
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, status int, err error) {
	if status >= 500 {
		h.log.ErrorContext(r.Context(), "request failed", "method", r.Method, "path", r.URL.Path, "err", err)
		writeJSON(w, status, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
