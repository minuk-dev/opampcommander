// Package application provides application services for applications.
package application

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
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/usecase"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

var _ usecase.ApplicationManageUsecase = (*Service)(nil)

// Service implements the ApplicationManageUsecase interface.
type Service struct {
	applicationUsecase agentport.ApplicationUsecase
	agentUsecase       agentport.AgentUsecase
	mapper             *helper.Mapper
	logger             *slog.Logger
}

// New creates a new application Service.
func New(
	applicationUsecase agentport.ApplicationUsecase,
	agentUsecase agentport.AgentUsecase,
	logger *slog.Logger,
) *Service {
	return &Service{
		applicationUsecase: applicationUsecase,
		agentUsecase:       agentUsecase,
		mapper:             helper.NewMapper(clock.RealClock{}, agentmodel.DefaultConnectionStaleness),
		logger:             logger,
	}
}

// GetApplication implements usecase.ApplicationManageUsecase.
func (s *Service) GetApplication(ctx context.Context, id string) (*v1.Application, error) {
	application, err := s.applicationUsecase.GetApplication(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get application: %w", err)
	}

	return mapApplicationToAPI(application), nil
}

// ListApplications implements usecase.ApplicationManageUsecase.
func (s *Service) ListApplications(
	ctx context.Context,
	options *applicationport.ListOptions,
) (*v1.ListResponse[v1.Application], error) {
	response, err := s.applicationUsecase.ListApplications(ctx, options.ToDomain())
	if err != nil {
		return nil, fmt.Errorf("failed to list applications: %w", err)
	}

	return &v1.ListResponse[v1.Application]{
		Kind:       v1.ApplicationKind,
		APIVersion: v1.APIVersion,
		Metadata: v1.ListMeta{
			Continue:           response.Continue,
			RemainingItemCount: response.RemainingItemCount,
		},
		Items: lo.Map(response.Items, func(application *agentmodel.Application, _ int) v1.Application {
			return *mapApplicationToAPI(application)
		}),
	}, nil
}

// ListAgentsByApplication implements usecase.ApplicationManageUsecase.
func (s *Service) ListAgentsByApplication(
	ctx context.Context,
	id string,
	options *applicationport.ListOptions,
) (*v1.ListResponse[v1.Agent], error) {
	application, err := s.applicationUsecase.GetApplication(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get application: %w", err)
	}

	page, err := helper.PaginateUUIDs(application.Status.AgentInstanceUIDs, options.ToDomain())
	if err != nil {
		return nil, fmt.Errorf("failed to paginate application agents: %w", err)
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

			return nil, fmt.Errorf("failed to get agent for application: %w", err)
		}

		items = append(items, *s.mapper.MapAgentToAPI(agent))
	}

	return items, nil
}
