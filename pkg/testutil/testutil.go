// Package testutil provides utility functions and types for testing.
package testutil

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"go.uber.org/goleak"
)

// Base is a base.
//
//nolint:containedctx
type Base struct {
	t   testing.TB
	ctx context.Context

	Logger       *slog.Logger
	Dependencies map[string]Dependency

	CacheDir string
}

// Dependency is an interface that represents a dependency in the test environment.
type Dependency interface {
	Name() string
	Configure(config any)
	Start()
	Stop()
	Info() map[string]string
}

// NewBase creates a new instance of Base.
func NewBase(tb testing.TB) *Base {
	tb.Helper()

	ctx, cancel := context.WithCancel(context.Background())

	tb.Cleanup(func() {
		goleak.VerifyNone(tb)
		cancel()
	})

	cacheDir, ok := os.LookupEnv("OPAMP_COMMANDER_TESTING_DIR")
	if !ok {
		cacheDir = tb.TempDir()
	}

	return &Base{
		t:            tb,
		ctx:          ctx,
		Logger:       slog.Default(),
		Dependencies: make(map[string]Dependency),
		CacheDir:     cacheDir,
	}
}
