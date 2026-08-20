package inmemory

import (
	"time"
)

const (
	// DefaultTTL is the age at which an untouched record is evicted. Set well above
	// [agentmodel.DefaultConnectionStaleness] so a record is never dropped while the
	// agent it describes could still be reported as connected.
	DefaultTTL = 30 * time.Minute
	// DefaultGCInterval is how often expired records are swept.
	DefaultGCInterval = 5 * time.Minute
)

// Config holds the tuning knobs of the store. Zero values fall back to the
// package defaults.
type Config struct {
	// TTL is the age at which an untouched record is evicted.
	TTL time.Duration
	// GCInterval is how often expired records are swept.
	GCInterval time.Duration
}

// effectiveTTL returns the configured TTL, or the default when unset.
func (c Config) effectiveTTL() time.Duration {
	if c.TTL <= 0 {
		return DefaultTTL
	}

	return c.TTL
}

// effectiveGCInterval returns the configured sweep interval, or the default when unset.
func (c Config) effectiveGCInterval() time.Duration {
	if c.GCInterval <= 0 {
		return DefaultGCInterval
	}

	return c.GCInterval
}
