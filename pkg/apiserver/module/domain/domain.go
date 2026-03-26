// Package domain provides the domain services module for the API server.
package domain

import (
	"context"
	"log/slog"

	"go.uber.org/fx"

	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
	domainservice "github.com/minuk-dev/opampcommander/internal/domain/service"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/module/helper"
)

// New creates a new module for domain services.
func New() fx.Option {
	components := []any{
		fx.Annotate(domainservice.NewConnectionService, fx.As(new(domainport.ConnectionUsecase))),
		provideAgentService,
		fx.Annotate(
			Identity[*domainservice.AgentService],
			fx.As(new(domainport.AgentUsecase)),
		),
		domainservice.NewAgentGroupService,
		fx.Annotate(
			Identity[*domainservice.AgentGroupService],
			fx.As(new(domainport.AgentGroupUsecase)),
			fx.As(new(domainport.AgentGroupRelatedUsecase)),
		),
		fx.Annotate(domainservice.NewAgentPackageService, fx.As(new(domainport.AgentPackageUsecase))),
		fx.Annotate(domainservice.NewAgentRemoteConfigService, fx.As(new(domainport.AgentRemoteConfigUsecase))),
		fx.Annotate(domainservice.NewCertificateService, fx.As(new(domainport.CertificateUsecase))),
		domainservice.NewServerService,
		fx.Annotate(
			Identity[*domainservice.ServerService],
			fx.As(new(domainport.ServerUsecase)),
			fx.As(new(domainport.ServerMessageUsecase)),
		),
		domainservice.NewServerIdentityService,
		fx.Annotate(
			Identity[*domainservice.ServerIdentityService],
			fx.As(new(domainport.ServerIdentityProvider)),
		),
		fx.Annotate(domainservice.NewAgentNotificationService, fx.As(new(domainport.AgentNotificationUsecase))),
		// RBAC domain services
		fx.Annotate(domainservice.NewUserService, fx.As(new(domainport.UserUsecase))),
		fx.Annotate(domainservice.NewRoleService, fx.As(new(domainport.RoleUsecase))),
		fx.Annotate(domainservice.NewPermissionService, fx.As(new(domainport.PermissionUsecase))),
		fx.Annotate(domainservice.NewUserRoleService, fx.As(new(domainport.UserRoleUsecase))),
		fx.Annotate(domainservice.NewRBACService, fx.As(new(domainport.RBACUsecase))),
		fx.Annotate(domainservice.NewOrgRoleMappingService, fx.As(new(domainport.OrgRoleMappingUsecase))),
		helper.AsRunner(Identity[*domainservice.AgentGroupService]),
		helper.AsRunner(Identity[*domainservice.ServerService]),
		helper.AsRunner(Identity[*domainservice.ServerIdentityService]),
	}

	return fx.Module(
		"domain",
		fx.Provide(components...),
		fx.Invoke(registerShutdownHooks),
	)
}

func provideAgentService(
	agentPersistencePort domainport.AgentPersistencePort,
	logger *slog.Logger,
	settings *config.ServerSettings,
) *domainservice.AgentService {
	// Apply default cache settings if not explicitly configured
	cacheSettings := settings.CacheSettings
	//nolint:exhaustruct // Intentionally comparing with zero value to check if not configured
	if cacheSettings == (config.CacheSettings{}) {
		cacheSettings = config.DefaultCacheSettings()
	}

	agentCacheSettings := cacheSettings.Agent

	return domainservice.NewAgentServiceWithConfig(
		agentPersistencePort,
		logger,
		domainservice.AgentCacheConfig{
			Enabled:     agentCacheSettings.Enabled,
			TTL:         agentCacheSettings.TTL,
			MaxCapacity: agentCacheSettings.MaxCapacity,
		},
	)
}

// registerShutdownHooks registers shutdown hooks for services with caches.
func registerShutdownHooks(
	lifecycle fx.Lifecycle,
	agentService *domainservice.AgentService,
	serverService *domainservice.ServerService,
) {
	lifecycle.Append(fx.Hook{
		OnStart: nil,
		OnStop: func(_ context.Context) error {
			agentService.Shutdown()
			serverService.Shutdown()

			return nil
		},
	})
}

// Identity is a generic function that returns the input value.
// It is a helper function to generate a function that returns the input value.
// It is used to provide a function as a interface.
func Identity[T any](a T) T {
	return a
}
