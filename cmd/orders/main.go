// Command orders is the composition root: the only place that knows which
// adapter implements which port.
//
// Right now everything is in memory. On step 14 you rewrite this file: config,
// choosing adapters, logging, metrics, tracing, graceful shutdown.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/s4bb4t/infra-layer-practice/internal/app/usecase"
	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/memory"
	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/system"
	transport "github.com/s4bb4t/infra-layer-practice/internal/presentation/http"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("exit", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store := memory.NewStore()
	deps := usecase.Deps{Tx: store, Orders: store, Outbox: store, Clock: system.Clock{}, IDs: system.UUIDv7{}}

	h := transport.NewHandler(transport.UseCases{
		Place:  usecase.NewPlaceOrder(deps),
		Pay:    usecase.NewPayOrder(deps),
		Cancel: usecase.NewCancelOrder(deps),
		Get:    usecase.NewGetOrder(store),
	}, log)

	srv := &http.Server{Addr: ":8080", Handler: h, ReadHeaderTimeout: 5 * time.Second}
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	log.Info("listening", "addr", srv.Addr)

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
