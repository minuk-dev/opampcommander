// Package testutil provides utility functions and types for testing.
package testutil

import (
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

// Base is a utility struct that provides common resources and utilities for testing.
type Base struct {
	t testing.TB

	Logger       *slog.Logger
	Dependencies map[string]Dependency

	CacheDir string

	// serverIDCounter hands out distinct ServerID suffixes for apiservers
	// spawned in the same test. Two servers sharing an ID collide on
	// ServerIdentityService.registerServer.
	serverIDCounter atomic.Uint64
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

	//exhaustruct:ignore
	return &Base{
		t:            tb,
		Logger:       slog.Default(),
		Dependencies: make(map[string]Dependency),
		CacheDir:     cacheDir,
	}
}

// nextServerID returns a unique server ID for this test, derived from the test
// name with a monotonic suffix so multiple apiservers in one test do not
// collide.
func (b *Base) nextServerID() string {
	b.t.Helper()

	return Identifier(b.t) + "-" + strconv.FormatUint(b.serverIDCounter.Add(1), 10)
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
