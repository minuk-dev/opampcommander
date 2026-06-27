// Package agentpackage provides the AgentPackageService for managing agent packages.
package agentpackage

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/samber/lo"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/helper"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/security"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var _ port.AgentPackageManageUsecase = (*Service)(nil)

// Service is a service for managing agent packages. It maps between the HTTP DTOs
// and the domain, resolves the acting user, and delegates all lifecycle rules
// (stamping, immutable-field preservation) to the domain AgentPackageUsecase.
type Service struct {
	agentpackageUsecase agentport.AgentPackageUsecase
	mapper              *helper.Mapper
	clock               clock.Clock
	logger              *slog.Logger
}

// NewAgentPackageService creates a new AgentPackageService.
func NewAgentPackageService(
	agentpackageUsecase agentport.AgentPackageUsecase,
	logger *slog.Logger,
) *Service {
	realClock := clock.NewRealClock()

	return &Service{
		agentpackageUsecase: agentpackageUsecase,
		mapper:              helper.NewMapper(realClock, 0),
		clock:               realClock,
		logger:              logger,
	}
}

// GetAgentPackage implements [port.AgentPackageManageUsecase].
func (a *Service) GetAgentPackage(
	ctx context.Context,
	namespace string,
	name string,
	options *port.GetOptions,
) (*v1.AgentPackage, error) {
	agentPackage, err := a.agentpackageUsecase.GetAgentPackage(ctx, namespace, name, options.ToDomain())
	if err != nil {
		return nil, fmt.Errorf("get agent package: %w", err)
	}

	return a.mapper.MapAgentPackageToAPI(agentPackage), nil
}

// ListAgentPackages implements [port.AgentPackageManageUsecase].
func (a *Service) ListAgentPackages(
	ctx context.Context,
	options *port.ListOptions,
) (*v1.ListResponse[v1.AgentPackage], error) {
	agentPackages, err := a.agentpackageUsecase.ListAgentPackages(ctx, options.ToDomain())
	if err != nil {
		return nil, fmt.Errorf("list agent packages: %w", err)
	}

	return &v1.ListResponse[v1.AgentPackage]{
		Kind:       v1.AgentPackageKind,
		APIVersion: v1.APIVersion,
		Metadata: v1.ListMeta{
			Continue:           agentPackages.Continue,
			RemainingItemCount: agentPackages.RemainingItemCount,
		},
		Items: lo.Map(agentPackages.Items, func(item *agentmodel.AgentPackage, _ int) v1.AgentPackage {
			return *a.mapper.MapAgentPackageToAPI(item)
		}),
	}, nil
}

// CreateAgentPackage implements [port.AgentPackageManageUsecase].
func (a *Service) CreateAgentPackage(
	ctx context.Context,
	apiModel *v1.AgentPackage,
) (*v1.AgentPackage, error) {
	domainModel := a.mapper.MapAPIToAgentPackage(apiModel)

	created, err := a.agentpackageUsecase.CreateAgentPackage(ctx, domainModel, a.actor(ctx))
	if err != nil {
		return nil, fmt.Errorf("create agent package: %w", err)
	}

	return a.mapper.MapAgentPackageToAPI(created), nil
}

// UpdateAgentPackage implements [port.AgentPackageManageUsecase].
func (a *Service) UpdateAgentPackage(
	ctx context.Context,
	namespace string,
	name string,
	agentPackage *v1.AgentPackage,
) (*v1.AgentPackage, error) {
	domainModel := a.mapper.MapAPIToAgentPackage(agentPackage)

	updated, err := a.agentpackageUsecase.UpdateAgentPackage(ctx, namespace, name, domainModel)
	if err != nil {
		return nil, fmt.Errorf("update agent package: %w", err)
	}

	return a.mapper.MapAgentPackageToAPI(updated), nil
}

// DeleteAgentPackage implements [port.AgentPackageManageUsecase].
func (a *Service) DeleteAgentPackage(
	ctx context.Context,
	namespace string,
	name string,
) error {
	err := a.agentpackageUsecase.DeleteAgentPackage(
		ctx, namespace, name, a.clock.Now(), a.actor(ctx),
	)
	if err != nil {
		return fmt.Errorf("delete agent package: %w", err)
	}

	return nil
}

// actor resolves the acting user from the request context, falling back to an
// anonymous identity (and logging) when none is present.
func (a *Service) actor(ctx context.Context) string {
	user, err := security.GetUser(ctx)
	if err != nil {
		a.logger.Warn("failed to get user from context", slog.String("error", err.Error()))

		user = security.NewAnonymousUser()
	}

	return user.String()
}
