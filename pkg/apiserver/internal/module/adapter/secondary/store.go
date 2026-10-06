package secondary

import (
	"context"

	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/store/cached"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/store/inmemory"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
)

func provideAgentStore(persistence agentport.AgentPersistencePort, settings *config.ServerSettings) *cached.AgentStore {
	cacheSettings := settings.CacheSettings
	if cacheSettings == (config.CacheSettings{}) {
		cacheSettings = config.DefaultCacheSettings()
	}

	return cached.NewAgentStore(persistence, cached.AgentCacheConfig{
		Enabled:     cacheSettings.Agent.Enabled,
		TTL:         cacheSettings.Agent.TTL,
		MaxCapacity: cacheSettings.Agent.MaxCapacity,
	})
}

func provideNotificationStore() *inmemory.NotificationStore {
	return inmemory.NewNotificationStore(agentservice.DefaultNotificationDispatchQueue)
}

func provideAgentGroupChangeStore() *inmemory.AgentGroupChangeStore {
	return inmemory.NewAgentGroupChangeStore(agentservice.ChangedAgentGroupBufferSize)
}

func registerStoreShutdown(lifecycle fx.Lifecycle, agents *cached.AgentStore, servers *cached.ServerStore) {
	lifecycle.Append(fx.Hook{
		OnStart: nil,
		OnStop: func(_ context.Context) error {
			agents.Shutdown()
			servers.Shutdown()

			return nil
		},
	})
}
