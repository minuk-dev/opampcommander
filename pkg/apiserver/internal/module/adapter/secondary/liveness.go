package secondary

import (
	"context"
	"fmt"
	"log/slog"

	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/liveness/failover"
	livenessinmemory "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/liveness/inmemory"
	livenessredis "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/liveness/redis"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/internal/module/helper"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

// NewLiveness provides the agent liveness fast tier.
//
// The node-local store is always provided: it is the default tier, and it stays the
// fallback when a shared one is configured. Redis is layered on top only when it is
// both enabled and meaningful — a standalone server keeps its whole state in
// process, so a shared tier there would add a network hop to reach data it already
// holds.
func NewLiveness(databaseType config.DatabaseType, livenessSettings config.LivenessSettings) fx.Option {
	options := []fx.Option{
		fx.Provide(
			newInMemoryLivenessStore,
			helper.AsRunner(identity[*livenessinmemory.Store]),
		),
	}

	if livenessSettings.Redis.Enabled && databaseType == config.DatabaseTypeMongoDB {
		options = append(options, fx.Provide(
			newRedisLivenessStore,
			newFailoverLivenessStore,
			fx.Annotate(identity[*failover.Store], fx.As(new(agentport.AgentLivenessPort))),
			helper.AsHealthIndicator(NewAgentLivenessHealthIndicator),
		), fx.Invoke(bindForcedFlush))

		return fx.Options(options...)
	}

	options = append(options, fx.Provide(
		fx.Annotate(identity[*livenessinmemory.Store], fx.As(new(agentport.AgentLivenessPort))),
	))

	if livenessSettings.Redis.Enabled {
		// Say so rather than quietly ignoring it: an operator who configured Redis
		// expects it to be in use, and silence here would look like Redis working.
		options = append(options, fx.Invoke(func(logger *slog.Logger) {
			logger.Warn("liveness redis is configured but ignored in standalone mode; " +
				"a standalone server keeps its whole state in process, so a shared tier " +
				"would only add a network hop to reach data it already holds")
		}))
	}

	return fx.Options(options...)
}

func newInMemoryLivenessStore() *livenessinmemory.Store {
	//exhaustruct:ignore // zero fields select the store's defaults
	return livenessinmemory.New(livenessinmemory.Config{}, clock.NewRealClock())
}

// newRedisLivenessStore builds the shared fast tier.
//
// Reaching Redis is deliberately not a startup requirement: the store is created
// from configuration alone and connects lazily, so a Redis that is down at boot
// costs extra database writes rather than a server that refuses to start. The
// lifecycle hook probes it once only to say so in the log.
func newRedisLivenessStore(
	settings *config.ServerSettings,
	logger *slog.Logger,
	lifecycle fx.Lifecycle,
) (*livenessredis.Store, error) {
	redisSettings := settings.LivenessSettings.Redis

	store, err := livenessredis.New(livenessredis.Config{
		Endpoints:      redisSettings.Endpoints,
		MasterName:     redisSettings.MasterName,
		Username:       redisSettings.Username,
		Password:       redisSettings.Password,
		DB:             redisSettings.DB,
		DialTimeout:    redisSettings.EffectiveDialTimeout(),
		CommandTimeout: redisSettings.EffectiveCommandTimeout(),
		TTL:            redisSettings.EffectiveTTL(),
		// The default namespace is right for a Redis dedicated to one deployment; a
		// shared one is not addressed here yet.
		KeyPrefix: livenessredis.DefaultKeyPrefix,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build the redis agent liveness store: %w", err)
	}

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			pingErr := store.Ping(ctx)
			if pingErr != nil {
				logger.Warn("redis agent liveness tier is unreachable at startup; "+
					"the server will keep working against the database",
					slog.String("error", pingErr.Error()),
				)

				return nil
			}

			logger.Info("redis agent liveness tier connected",
				slog.Any("endpoints", redisSettings.Endpoints),
				slog.Duration("ttl", redisSettings.EffectiveTTL()),
			)

			return nil
		},
		OnStop: func(context.Context) error {
			closeErr := store.Close()
			if closeErr != nil {
				logger.Warn("failed to close the redis agent liveness tier",
					slog.String("error", closeErr.Error()))
			}

			return nil
		},
	})

	return store, nil
}

// newFailoverLivenessStore puts the circuit breaker between the shared tier and the
// node-local one, so an unreachable Redis costs database writes rather than failed
// requests.
func newFailoverLivenessStore(
	primary *livenessredis.Store,
	fallback *livenessinmemory.Store,
	logger *slog.Logger,
) *failover.Store {
	//exhaustruct:ignore // zero fields select the breaker's defaults
	return failover.New(primary, fallback, failover.Config{}, clock.NewRealClock(), logger)
}

// bindForcedFlush makes a tripping breaker force a write-behind flush.
//
// The wiring is a separate step because the two halves depend on each other: the
// flusher reads through the liveness port, so it cannot be a constructor argument
// of the store that provides it.
func bindForcedFlush(
	store *failover.Store,
	flusher *agentservice.AgentLivenessFlusher,
	logger *slog.Logger,
	lifecycle fx.Lifecycle,
) {
	// A forced flush outlives the call that tripped the breaker, so shutdown waits
	// for it rather than exiting mid-write.
	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error { return nil },
		OnStop: func(context.Context) error {
			store.Wait()

			return nil
		},
	})

	store.OnTrip(func(ctx context.Context) {
		written, err := flusher.Flush(ctx)
		if err != nil {
			logger.Warn("forced agent liveness flush failed after the circuit breaker tripped",
				slog.String("error", err.Error()))

			return
		}

		logger.Info("forced agent liveness flush after the circuit breaker tripped",
			slog.Int("agents", written))
	})
}
