// Package testutil provides utility functions and types for testing.
package testutil

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"go.uber.org/goleak"
)

// Base is a base
type Base struct {
	t   testing.TB
	ctx context.Context

	Logger       *slog.Logger
	Dependencies map[string]Dependency

	CacheDir string
}

type Dependency interface {
	Name() string
	Configure(config any)
	Start()
	Stop()
	Info() map[string]string
}

// NewBase creates a new instance of Base.
func NewBase(t testing.TB) *Base {
	ctx, cancel := context.WithCancel(context.Background())

	t.Cleanup(func() {
		goleak.VerifyNone(t)
		cancel()
	})

	cacheDir, ok := os.LookupEnv("OPAMP_COMMANDER_TESTING_DIR")
	if !ok {
		cacheDir = t.TempDir()
	}

	return &Base{
		t:            t,
		ctx:          ctx,
		Logger:       slog.Default(),
		Dependencies: make(map[string]Dependency),
		CacheDir:     cacheDir,
	}
}
