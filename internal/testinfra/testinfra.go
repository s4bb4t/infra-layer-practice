// Package testinfra starts real infrastructure in Docker for integration tests.
//
// Every Start* function returns connection details and registers nothing
// global: the caller decides the container lifetime (usually TestMain:
// one container per test package).
package testinfra

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/testcontainers/testcontainers-go"
)

// Container is a started container plus the address tests need.
type Container struct {
	testcontainers.Container
	// URI is a ready-to-use connection string (DSN, broker address, ...).
	URI string
}

// Main is a helper for TestMain: it runs start, then the tests, then stops
// the container. Panics on start errors: there is no *testing.T in TestMain.
func Main(m *testing.M, start func(ctx context.Context) (*Container, error), set func(*Container)) {
	ctx := context.Background()
	c, err := start(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "start container:", err)
		os.Exit(1)
	}
	set(c)
	code := m.Run()
	_ = c.Terminate(ctx)
	os.Exit(code)
}
