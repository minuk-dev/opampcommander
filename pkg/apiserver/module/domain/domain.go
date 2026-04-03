// Package domain provides the domain services module for the API server.
package domain

import (
	"context"
	"log/slog"

	"go.uber.org/fx"

	agentport "github.com/minuk-dev/opampcommander/internal/domain/agent/port"
	agentservice "github.com/minuk-dev/opampcommander/internal/domain/agent/service"
	userport "github.com/minuk-dev/opampcommander/internal/domain/user/port"
	userservice "github.com/minuk-dev/opampcommander/internal/domain/user/service"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/module/helper"
)

// New creates a new module for domain services.
func New() fx.Option {
	components := []any{
		fx.Annotate(agentservice.NewConnectionService, fx.As(new(agentport.ConnectionUsecase))),
		provideAgentService,
		fx.Annotate(
			Identity[*agentservice.AgentService],
			fx.As(new(agentport.AgentUsecase)),
		),
		agentservice.NewAgentGroupService,
		fx.Annotate(
			Identity[*agentservice.AgentGroupService],
			fx.As(new(agentport.AgentGroupUsecase)),
			fx.As(new(agentport.AgentGroupRelatedUsecase)),
		),
		fx.Annotate(agentservice.NewAgentPackageService, fx.As(new(agentport.AgentPackageUsecase))),
		fx.Annotate(agentservice.NewAgentRemoteConfigService, fx.As(new(agentport.AgentRemoteConfigUsecase))),
		fx.Annotate(agentservice.NewCertificateService, fx.As(new(agentport.CertificateUsecase))),
		agentservice.NewServerService,
		fx.Annotate(
			Identity[*agentservice.ServerService],
			fx.As(new(agentport.ServerUsecase)),
			fx.As(new(agentport.ServerMessageUsecase)),
		),
		agentservice.NewServerIdentityService,
		fx.Annotate(
			Identity[*agentservice.ServerIdentityService],
			fx.As(new(agentport.ServerIdentityProvider)),
		),
		fx.Annotate(agentservice.NewAgentNotificationService, fx.As(new(agentport.AgentNotificationUsecase))),
		// RBAC domain services
		fx.Annotate(userservice.NewUserService, fx.As(new(userport.UserUsecase))),
		fx.Annotate(userservice.NewRoleService, fx.As(new(userport.RoleUsecase))),
		fx.Annotate(userservice.NewPermissionService, fx.As(new(userport.PermissionUsecase))),
		fx.Annotate(userservice.NewUserRoleService, fx.As(new(userport.UserRoleUsecase))),
		fx.Annotate(userservice.NewRBACService, fx.As(new(userport.RBACUsecase))),
		fx.Annotate(userservice.NewOrgRoleMappingService, fx.As(new(userport.OrgRoleMappingUsecase))),
		helper.AsRunner(Identity[*agentservice.AgentGroupService]),
		helper.AsRunner(Identity[*agentservice.ServerService]),
		helper.AsRunner(Identity[*agentservice.ServerIdentityService]),
	}

	return fx.Module(
		"domain",
		fx.Provide(components...),
		fx.Invoke(registerShutdownHooks),
	)
}

func provideAgentService(
	agentPersistencePort agentport.AgentPersistencePort,
	logger *slog.Logger,
	settings *config.ServerSettings,
) *agentservice.AgentService {
	// Apply default cache settings if not explicitly configured
	cacheSettings := settings.CacheSettings
	//nolint:exhaustruct // Intentionally comparing with zero value to check if not configured
	if cacheSettings == (config.CacheSettings{}) {
		cacheSettings = config.DefaultCacheSettings()
	}

	agentCacheSettings := cacheSettings.Agent

	return agentservice.NewAgentServiceWithConfig(
		agentPersistencePort,
		logger,
		agentservice.AgentCacheConfig{
			Enabled:     agentCacheSettings.Enabled,
			TTL:         agentCacheSettings.TTL,
			MaxCapacity: agentCacheSettings.MaxCapacity,
		},
	)
}

// registerShutdownHooks registers shutdown hooks for services with caches.
func registerShutdownHooks(
	lifecycle fx.Lifecycle,
	agentService *agentservice.AgentService,
	serverService *agentservice.ServerService,
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
