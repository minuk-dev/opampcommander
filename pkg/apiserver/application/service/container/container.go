// Package container provides application services for containers.
package container

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"k8s.io/utils/clock"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/helper"
	applicationport "github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

var _ applicationport.ContainerManageUsecase = (*Service)(nil)

// Service implements the ContainerManageUsecase interface.
type Service struct {
	containerUsecase agentport.ContainerUsecase
	agentUsecase     agentport.AgentUsecase
	mapper           *helper.Mapper
	logger           *slog.Logger
}

// New creates a new container application Service.
func New(
	containerUsecase agentport.ContainerUsecase,
	agentUsecase agentport.AgentUsecase,
	logger *slog.Logger,
) *Service {
	return &Service{
		containerUsecase: containerUsecase,
		agentUsecase:     agentUsecase,
		mapper:           helper.NewMapper(clock.RealClock{}, agentmodel.DefaultConnectionStaleness),
		logger:           logger,
	}
}

// GetContainer implements applicationport.ContainerManageUsecase.
func (s *Service) GetContainer(ctx context.Context, id string) (*v1.Container, error) {
	container, err := s.containerUsecase.GetContainer(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get container: %w", err)
	}

	return mapContainerToAPI(container), nil
}

// ListContainers implements applicationport.ContainerManageUsecase.
func (s *Service) ListContainers(
	ctx context.Context,
	options *applicationport.ListOptions,
) (*v1.ListResponse[v1.Container], error) {
	response, err := s.containerUsecase.ListContainers(ctx, options.ToDomain())
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	return &v1.ListResponse[v1.Container]{
		Kind:       v1.ContainerKind,
		APIVersion: v1.APIVersion,
		Metadata: v1.ListMeta{
			Continue:           response.Continue,
			RemainingItemCount: response.RemainingItemCount,
		},
		Items: lo.Map(response.Items, func(container *agentmodel.Container, _ int) v1.Container {
			return *mapContainerToAPI(container)
		}),
	}, nil
}

// ListAgentsByContainer implements applicationport.ContainerManageUsecase.
func (s *Service) ListAgentsByContainer(
	ctx context.Context,
	id string,
	options *applicationport.ListOptions,
) (*v1.ListResponse[v1.Agent], error) {
	container, err := s.containerUsecase.GetContainer(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get container: %w", err)
	}

	page, err := helper.PaginateUUIDs(container.Status.AgentInstanceUIDs, options.ToDomain())
	if err != nil {
		return nil, fmt.Errorf("failed to paginate container agents: %w", err)
	}

	items, err := s.resolveAgents(ctx, page.Items)
	if err != nil {
		return nil, err
	}

	return &v1.ListResponse[v1.Agent]{
		Kind:       v1.AgentKind,
		APIVersion: v1.APIVersion,
		Metadata: v1.ListMeta{
			Continue:           page.Continue,
			RemainingItemCount: page.RemainingItemCount,
		},
		Items: items,
	}, nil
}

// resolveAgents fetches and maps the agents for the given instance UIDs, skipping
// any that have since been removed (the association is a best-effort discovery
// snapshot).
func (s *Service) resolveAgents(ctx context.Context, instanceUIDs []uuid.UUID) ([]v1.Agent, error) {
	items := make([]v1.Agent, 0, len(instanceUIDs))

	for _, instanceUID := range instanceUIDs {
		agent, err := s.agentUsecase.GetAgent(ctx, instanceUID)
		if err != nil {
			if errors.Is(err, model.ErrResourceNotExist) {
				continue
			}

			return nil, fmt.Errorf("failed to get agent for container: %w", err)
		}

		items = append(items, *s.mapper.MapAgentToAPI(agent))
	}

	return items, nil
}
