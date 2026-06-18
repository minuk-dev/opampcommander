package secondary

import (
	"context"

	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/inmemory"
	applicationport "github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	userport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/internal/module/helper"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/management/healthcheck"
)

// NewInMemory provides the in-memory persistence adapters used in standalone
// (single-node) mode, replacing the MongoDB client/database and repositories.
//
// Three repositories are provided as concrete singletons in addition to their
// port bindings, because other repositories depend on the concrete type and
// must share the very same store instance:
//   - *inmemory.AgentRepository backs the agent-group statistics,
//   - *inmemory.RoleRepository and *inmemory.UserRepository back the user-role
//     cross-entity lookups.
func NewInMemory() fx.Option {
	return fx.Options(
		// Concrete singletons shared across repositories.
		fx.Provide(
			inmemory.NewAgentRepository,
			inmemory.NewRoleRepository,
			inmemory.NewUserRepository,
		),
		fx.Provide(
			// Bind the shared concrete singletons to their persistence ports.
			fx.Annotate(identity[*inmemory.AgentRepository], fx.As(new(agentport.AgentPersistencePort))),
			fx.Annotate(identity[*inmemory.RoleRepository], fx.As(new(userport.RolePersistencePort))),
			fx.Annotate(identity[*inmemory.UserRepository], fx.As(new(userport.UserPersistencePort))),

			// Health + transaction.
			helper.AsHealthIndicator(NewInMemoryHealthIndicator),
			fx.Annotate(inmemory.NewTransactionRunner, fx.As(new(applicationport.TransactionRunner))),

			// Agent-domain repositories.
			fx.Annotate(inmemory.NewAgentGroupRepository, fx.As(new(agentport.AgentGroupPersistencePort))),
			fx.Annotate(inmemory.NewServerRepository, fx.As(new(agentport.ServerPersistencePort))),
			fx.Annotate(inmemory.NewAgentPackageRepository, fx.As(new(agentport.AgentPackagePersistencePort))),
			fx.Annotate(inmemory.NewNamespaceRepository, fx.As(new(agentport.NamespacePersistencePort))),
			fx.Annotate(inmemory.NewAgentRemoteConfigRepository, fx.As(new(agentport.AgentRemoteConfigPersistencePort))),
			fx.Annotate(inmemory.NewEndpointRepository, fx.As(new(agentport.EndpointPersistencePort))),
			fx.Annotate(inmemory.NewCertificateRepository, fx.As(new(agentport.CertificatePersistencePort))),
			fx.Annotate(inmemory.NewHostRepository, fx.As(new(agentport.HostPersistencePort))),
			fx.Annotate(inmemory.NewContainerRepository, fx.As(new(agentport.ContainerPersistencePort))),

			// RBAC repositories.
			fx.Annotate(inmemory.NewPermissionRepository, fx.As(new(userport.PermissionPersistencePort))),
			fx.Annotate(inmemory.NewUserRoleRepository, fx.As(new(userport.UserRolePersistencePort))),
			fx.Annotate(inmemory.NewRoleBindingRepository, fx.As(new(userport.RoleBindingPersistencePort))),
		),
	)
}

// identity returns its argument. It lets FX expose a concrete singleton under an
// interface type without re-running the constructor (which would create a second
// instance with its own, unshared store).
func identity[T any](value T) T {
	return value
}

// InMemoryHealthIndicator is a health indicator for the in-memory store. It is
// always healthy and ready, since the store lives in process memory.
type InMemoryHealthIndicator struct{}

var _ healthcheck.HealthIndicator = (*InMemoryHealthIndicator)(nil)

// NewInMemoryHealthIndicator creates a new InMemoryHealthIndicator.
func NewInMemoryHealthIndicator() *InMemoryHealthIndicator {
	return &InMemoryHealthIndicator{}
}

// Name returns the name of the health indicator.
func (*InMemoryHealthIndicator) Name() string {
	return "InMemory"
}

// Readiness always reports ready.
func (*InMemoryHealthIndicator) Readiness(context.Context) healthcheck.Readiness {
	return healthcheck.Readiness{
		Ready:  true,
		Reason: "",
	}
}

// Health always reports healthy.
func (*InMemoryHealthIndicator) Health(context.Context) healthcheck.Health {
	return healthcheck.Health{
		Healthy: true,
		Reason:  "in-memory store is always healthy",
	}
}
