package config

import (
	"errors"
	"fmt"
	"time"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
)

// Errors reported by [LivenessSettings.Validate].
var (
	// ErrLivenessCadenceTooSlow indicates a write-behind cadence that would let the
	// stored agent document age past the connection-staleness window.
	ErrLivenessCadenceTooSlow = errors.New(
		"liveness write-behind cadence would let stored agents age past the connection staleness window")
	// ErrLivenessRedisNoEndpoints indicates the Redis fast tier is enabled with no
	// endpoint to reach it at.
	ErrLivenessRedisNoEndpoints = errors.New("liveness redis is enabled but no endpoints are configured")
	// ErrLivenessRedisTTLTooShort indicates a Redis key TTL at or below the
	// connection-staleness window, which would expire a live agent's record.
	ErrLivenessRedisTTLTooShort = errors.New(
		"liveness redis ttl must be longer than the connection staleness window")
	// ErrLivenessRedisTimeoutInvalid indicates a non-positive Redis timeout.
	ErrLivenessRedisTimeoutInvalid = errors.New("liveness redis timeouts must be positive")
)

const (
	defaultLivenessFlushInterval       = 30 * time.Second
	defaultLivenessFlushStaleAfter     = 30 * time.Second
	defaultLivenessFlushBatchSize      = 2000
	defaultLivenessPersistThrottle     = 10 * time.Second
	defaultLivenessRedisDialTimeout    = 2 * time.Second
	defaultLivenessRedisCommandTimeout = 200 * time.Millisecond
	defaultLivenessRedisTTL            = 120 * time.Second

	// livenessFlushHeadroomDivisor sets how much of the connection-staleness window a
	// flush interval must leave unused: one part in this many.
	livenessFlushHeadroomDivisor = 3
)

// LivenessSettings configures the agent liveness fast tier — the fields a heartbeat
// refreshes every few seconds, which the server absorbs off the primary datastore
// and writes back on a slow cadence.
//
// The fast tier is a pure accelerator. With Redis disabled (the default) the server
// behaves exactly as it did before it existed: a node-local record per agent and a
// short per-message write throttle.
type LivenessSettings struct {
	// Redis configures the shared fast tier. Disabled by default.
	Redis RedisLivenessSettings `mapstructure:"redis"`

	// FlushInterval is how often observations absorbed by the fast tier are written
	// through to the durable store.
	// Default: 30s
	FlushInterval time.Duration `mapstructure:"flushInterval"`

	// FlushStaleAfter is how far behind a stored agent document must fall before the
	// write-behind flush claims it.
	//
	// It pairs with FlushInterval, and it is their sum that matters: a stored
	// document goes stale by up to FlushInterval + FlushStaleAfter, because the flush
	// only claims documents already FlushStaleAfter behind and only looks every
	// FlushInterval. That sum has to stay inside the staleness window, or the
	// datastore's own connected-agent filter and the agent-group connected counts —
	// neither of which any read-side overlay can reach — start reporting live agents
	// as disconnected.
	// Default: 30s
	FlushStaleAfter time.Duration `mapstructure:"flushStaleAfter"`

	// FlushBatchSize bounds how many agents one flush cycle writes.
	// Default: 2000
	FlushBatchSize int `mapstructure:"flushBatchSize"`

	// PersistThrottle is the minimum interval between write-throughs of an agent
	// whose only change is that it is still alive.
	//
	// Leave it unset: with no shared fast tier it defaults to 10s, where it is the
	// only thing keeping the stored document current, and with one it relaxes to the
	// staleness budget, handing the routine write path to the write-behind flush.
	// Setting it explicitly overrides both — but it is capped at the budget either
	// way, for the same reason the flush cadence is.
	PersistThrottle time.Duration `mapstructure:"persistThrottle"`
}

// RedisLivenessSettings configures the shared Redis fast tier.
//
// Redis is optional and never required: absent, misconfigured, or down, the server
// keeps working on the node-local tier and the durable store, just with more
// datastore writes.
type RedisLivenessSettings struct {
	// Enabled opts into the shared fast tier. Default: false.
	Enabled bool `mapstructure:"enabled"`
	// Endpoints are the Redis addresses. One address is a single server; several
	// address a cluster. Combine with MasterName to address a Sentinel set.
	Endpoints []string `mapstructure:"endpoints"`
	// MasterName selects Sentinel mode and names the monitored master.
	MasterName string `mapstructure:"masterName"`
	// Username and Password authenticate the connection.
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	// DB is the Redis logical database index. Ignored in cluster mode.
	DB int `mapstructure:"db"`
	// TLS enables a TLS connection.
	TLS bool `mapstructure:"tls"`
	// DialTimeout bounds establishing a connection. Default: 2s
	DialTimeout time.Duration `mapstructure:"dialTimeout"`
	// CommandTimeout bounds a single command. Kept short deliberately: this is a
	// fast path, and a slow Redis must degrade to the durable store rather than
	// hold up agent message processing. Default: 200ms
	CommandTimeout time.Duration `mapstructure:"commandTimeout"`
	// TTL is how long a liveness record survives without being refreshed. It must
	// outlive the connection-staleness window, or a live agent's record would expire
	// while the agent is still considered connected. Default: 120s
	TTL time.Duration `mapstructure:"ttl"`
}

// DefaultLivenessSettings returns the default liveness settings: no shared fast
// tier, which reproduces the server's behaviour before one existed.
func DefaultLivenessSettings() LivenessSettings {
	return LivenessSettings{
		Redis: RedisLivenessSettings{
			Enabled:        false,
			Endpoints:      nil,
			MasterName:     "",
			Username:       "",
			Password:       "",
			DB:             0,
			TLS:            false,
			DialTimeout:    defaultLivenessRedisDialTimeout,
			CommandTimeout: defaultLivenessRedisCommandTimeout,
			TTL:            defaultLivenessRedisTTL,
		},
		FlushInterval:   defaultLivenessFlushInterval,
		FlushStaleAfter: defaultLivenessFlushStaleAfter,
		FlushBatchSize:  defaultLivenessFlushBatchSize,
		PersistThrottle: 0,
	}
}

// MaxLivenessStalenessBudget returns the largest staleness the write-behind path may
// allow a stored agent document to reach.
//
// It is the budget every cadence knob shares: the flush interval plus its cutoff, and
// the message-path write-through throttle, each have to fit inside it.
func MaxLivenessStalenessBudget() time.Duration {
	staleness := agentmodel.DefaultConnectionStaleness

	return staleness - staleness/livenessFlushHeadroomDivisor
}

// EffectiveFlushInterval returns the configured flush interval, or the default when
// unset.
func (s LivenessSettings) EffectiveFlushInterval() time.Duration {
	if s.FlushInterval <= 0 {
		return defaultLivenessFlushInterval
	}

	return s.FlushInterval
}

// EffectiveFlushStaleAfter returns the configured write-behind cutoff, or the default
// when unset.
func (s LivenessSettings) EffectiveFlushStaleAfter() time.Duration {
	if s.FlushStaleAfter <= 0 {
		return defaultLivenessFlushStaleAfter
	}

	return s.FlushStaleAfter
}

// EffectivePersistThrottle returns the minimum interval between write-throughs of a
// still-alive agent, on the message path.
//
// With a shared fast tier the write-behind flush owns the routine write path, so the
// throttle relaxes to the staleness budget and heartbeat-only messages stop rewriting
// documents. Without one the throttle stays short, because it is then the only thing
// keeping the stored document current.
//
// Either way it is capped at the budget: it is how long an agent may keep
// heartbeating without its stored document being refreshed, so letting it exceed the
// budget has the same effect as an overlong flush cadence.
func (s LivenessSettings) EffectivePersistThrottle() time.Duration {
	throttle := s.PersistThrottle
	if throttle <= 0 {
		throttle = defaultLivenessPersistThrottle
		if s.Redis.Enabled {
			throttle = MaxLivenessStalenessBudget()
		}
	}

	return min(throttle, MaxLivenessStalenessBudget())
}

// Validate reports configuration that cannot work as written, so the server fails at
// startup rather than in a degraded state nobody notices.
func (s LivenessSettings) Validate() error {
	cadence := s.EffectiveFlushInterval() + s.EffectiveFlushStaleAfter()
	if cadence > MaxLivenessStalenessBudget() {
		return fmt.Errorf(
			"%w: flushInterval %s + flushStaleAfter %s = %s, but %s is the budget against a %s window",
			ErrLivenessCadenceTooSlow,
			s.EffectiveFlushInterval(), s.EffectiveFlushStaleAfter(), cadence,
			MaxLivenessStalenessBudget(), agentmodel.DefaultConnectionStaleness)
	}

	if !s.Redis.Enabled {
		return nil
	}

	return s.Redis.validate()
}

// EffectiveDialTimeout returns the configured dial timeout, or the default when unset.
func (s RedisLivenessSettings) EffectiveDialTimeout() time.Duration {
	if s.DialTimeout <= 0 {
		return defaultLivenessRedisDialTimeout
	}

	return s.DialTimeout
}

// EffectiveCommandTimeout returns the configured command timeout, or the default
// when unset.
func (s RedisLivenessSettings) EffectiveCommandTimeout() time.Duration {
	if s.CommandTimeout <= 0 {
		return defaultLivenessRedisCommandTimeout
	}

	return s.CommandTimeout
}

// EffectiveTTL returns the configured record TTL, or the default when unset.
func (s RedisLivenessSettings) EffectiveTTL() time.Duration {
	if s.TTL <= 0 {
		return defaultLivenessRedisTTL
	}

	return s.TTL
}

func (s RedisLivenessSettings) validate() error {
	if len(s.Endpoints) == 0 {
		return ErrLivenessRedisNoEndpoints
	}

	if s.EffectiveTTL() <= agentmodel.DefaultConnectionStaleness {
		return fmt.Errorf("%w: %s configured, must exceed %s",
			ErrLivenessRedisTTLTooShort, s.EffectiveTTL(), agentmodel.DefaultConnectionStaleness)
	}

	if s.DialTimeout < 0 || s.CommandTimeout < 0 {
		return fmt.Errorf("%w: dialTimeout=%s commandTimeout=%s",
			ErrLivenessRedisTimeoutInvalid, s.DialTimeout, s.CommandTimeout)
	}

	return nil
}
