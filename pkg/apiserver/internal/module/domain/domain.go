// Package domain provides the domain services module for the API server.
package domain

import (
	"context"
	"log/slog"

	"go.uber.org/fx"
	"k8s.io/utils/clock"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/reconcile"
	userport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user/port"
	userservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user/service"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/internal/module/helper"
)

// New creates a new module for domain services.
//
//nolint:funlen // DI wiring: a flat list of service providers/annotations.
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
		fx.Annotate(agentservice.NewNamespaceService, fx.As(new(agentport.NamespaceUsecase))),
		fx.Annotate(provideHostService, fx.As(new(agentport.HostUsecase))),
		fx.Annotate(provideContainerService, fx.As(new(agentport.ContainerUsecase))),
		fx.Annotate(agentservice.NewAgentRemoteConfigService, fx.As(new(agentport.AgentRemoteConfigUsecase))),
		fx.Annotate(agentservice.NewEndpointService, fx.As(new(agentport.EndpointUsecase))),
		fx.Annotate(agentservice.NewEndpointMetricsService, fx.As(new(agentport.EndpointMetricsUsecase))),
		fx.Annotate(agentservice.NewEndpointDetectionService, fx.As(new(agentport.EndpointDetectionUsecase))),
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
		agentservice.NewAgentNotificationService,
		fx.Annotate(
			Identity[*agentservice.AgentNotificationService],
			fx.As(new(agentport.AgentNotificationUsecase)),
		),
		// RBAC domain services
		fx.Annotate(userservice.NewUserService, fx.As(new(userport.UserUsecase))),
		fx.Annotate(userservice.NewRoleService, fx.As(new(userport.RoleUsecase))),
		fx.Annotate(userservice.NewPermissionService, fx.As(new(userport.PermissionUsecase))),
		fx.Annotate(userservice.NewUserRoleService, fx.As(new(userport.UserRoleUsecase))),
		fx.Annotate(userservice.NewRoleBindingService, fx.As(new(userport.RoleBindingUsecase))),
		provideRBACService,
		fx.Annotate(
			Identity[*userservice.RBACService],
			fx.As(new(userport.RBACUsecase)),
		),
		helper.AsRunner(Identity[*agentservice.AgentGroupService]),
		helper.AsRunner(Identity[*agentservice.ServerService]),
		helper.AsRunner(Identity[*agentservice.ServerIdentityService]),
		helper.AsRunner(Identity[*agentservice.AgentNotificationService]),

		// Generic reconcile registry: each Reconciler is collected into the "reconcilers"
		// group and indexed by reconcile.NewService. A new reconcilable kind plugs in by
		// adding one AsReconciler line here.
		helper.AsReconciler(agentservice.NewAgentRemoteConfigReconciler),
		helper.AsReconciler(agentservice.NewAgentGroupReconciler),
		helper.AsReconciler(agentservice.NewAgentReconciler),
		fx.Annotate(
			reconcile.NewService,
			fx.ParamTags(`group:"reconcilers"`),
		),
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
		settings.BootstrapSettings.DefaultNamespace,
	)
}

// provideHostService builds the host domain service with the real clock so
// discovery timestamps (FirstSeenAt/LastSeenAt) use wall-clock time.
func provideHostService(
	hostPersistencePort agentport.HostPersistencePort,
) *agentservice.HostService {
	return agentservice.NewHostService(hostPersistencePort, clock.RealClock{})
}

// provideContainerService builds the container domain service with the real clock.
func provideContainerService(
	containerPersistencePort agentport.ContainerPersistencePort,
) *agentservice.ContainerService {
	return agentservice.NewContainerService(containerPersistencePort, clock.RealClock{})
}

// provideRBACService builds the RBAC domain service, sourcing the built-in
// default role/namespace identifiers from configuration.
func provideRBACService(
	rbacEnforcerPort userport.RBACEnforcerPort,
	roleBindingPersistencePort userport.RoleBindingPersistencePort,
	rolePersistencePort userport.RolePersistencePort,
	permissionPersistencePort userport.PermissionPersistencePort,
	userPersistencePort userport.UserPersistencePort,
	logger *slog.Logger,
	settings *config.ServerSettings,
) *userservice.RBACService {
	return userservice.NewRBACService(
		rbacEnforcerPort,
		roleBindingPersistencePort,
		rolePersistencePort,
		permissionPersistencePort,
		userPersistencePort,
		logger,
		settings.BootstrapSettings.DefaultRole,
		settings.BootstrapSettings.DefaultNamespace,
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
