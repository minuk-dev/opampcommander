// Package testutil provides utility functions and types for testing.
package testutil

import (
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/opencontainers/go-digest"
)

// Base is a utility struct that provides common resources and utilities for testing.
type Base struct {
	t testing.TB

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

	cacheDir, ok := os.LookupEnv("OPAMP_COMMANDER_TESTING_DIR")
	if !ok {
		cacheDir = tb.TempDir()
	}

	return &Base{
		t:            tb,
		Logger:       slog.Default(),
		Dependencies: make(map[string]Dependency),
		CacheDir:     cacheDir,
	}
}

// Identifier can be used as a name of host, container, image, volume, network, etc.
func Identifier(t testing.TB) string {
	s := t.Name()
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ToLower(s)
	s = "opamp-" + s
	if len(s) > 76 {
		s = "opamp-" + digest.SHA256.FromString(t.Name()).Encoded()
	}
	return s
}

// TestLogWriter is an io.Writer that writes to the test log.
// This is useful for capturing logs in tests, especially when working with
// libraries that write to io.Writer (e.g., slog.TextHandler).
type TestLogWriter struct {
	T testing.TB
}

// Write implements io.Writer by logging to the test.
func (w TestLogWriter) Write(p []byte) (int, error) {
	w.T.Helper()
	w.T.Log(string(p))

	return len(p), nil
}
