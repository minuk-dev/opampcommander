// Package testutil provides utility functions and types for testing.
package testutil

import (
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"os"
	"strings"
	"testing"
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
func Identifier(tb testing.TB) string {
	tb.Helper()

	const maxIdentifierLen = 76

	name := tb.Name()
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ToLower(name)

	name = "opamp-" + name
	if len(name) > maxIdentifierLen {
		hash := sha256.Sum256([]byte(tb.Name()))
		name = "opamp-" + hex.EncodeToString(hash[:])
	}

	return name
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
