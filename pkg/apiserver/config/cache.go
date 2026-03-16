package config

import "time"

// CacheSettings holds the configuration for in-memory caching.
type CacheSettings struct {
	// Agent cache settings
	Agent AgentCacheSettings `mapstructure:"agent"`
}

// AgentCacheSettings holds the configuration for agent cache.
type AgentCacheSettings struct {
	// Enabled indicates whether the agent cache is enabled.
	// Default: true
	Enabled bool `mapstructure:"enabled"`
	// TTL is the time-to-live for cache entries.
	// Default: 30s
	TTL time.Duration `mapstructure:"ttl"`
	// MaxCapacity is the maximum number of items in the cache.
	// When exceeded, least recently used items are evicted.
	// Default: 1000
	MaxCapacity int64 `mapstructure:"maxCapacity"`
}

const (
	defaultAgentCacheTTL         = 30 * time.Second
	defaultAgentCacheMaxCapacity = 1000
)

// DefaultCacheSettings returns the default cache settings.
func DefaultCacheSettings() CacheSettings {
	return CacheSettings{
		Agent: AgentCacheSettings{
			Enabled:     true,
			TTL:         defaultAgentCacheTTL,
			MaxCapacity: defaultAgentCacheMaxCapacity,
		},
	}
}
