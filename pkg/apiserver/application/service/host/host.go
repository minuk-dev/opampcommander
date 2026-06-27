// Package host provides application services for hosts.
package host

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

var _ usecase.HostManageUsecase = (*Service)(nil)

// Service implements the HostManageUsecase interface.
type Service struct {
	hostUsecase  agentport.HostUsecase
	agentUsecase agentport.AgentUsecase
	mapper       *helper.Mapper
	logger       *slog.Logger
}

// New creates a new host application Service.
func New(
	hostUsecase agentport.HostUsecase,
	agentUsecase agentport.AgentUsecase,
	logger *slog.Logger,
) *Service {
	return &Service{
		hostUsecase:  hostUsecase,
		agentUsecase: agentUsecase,
		mapper:       helper.NewMapper(clock.RealClock{}, agentmodel.DefaultConnectionStaleness),
		logger:       logger,
	}
}

// GetHost implements usecase.HostManageUsecase.
func (s *Service) GetHost(ctx context.Context, id string) (*v1.Host, error) {
	host, err := s.hostUsecase.GetHost(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get host: %w", err)
	}

	return mapHostToAPI(host), nil
}

// ListHosts implements usecase.HostManageUsecase.
func (s *Service) ListHosts(
	ctx context.Context,
	options *applicationport.ListOptions,
) (*v1.ListResponse[v1.Host], error) {
	response, err := s.hostUsecase.ListHosts(ctx, options.ToDomain())
	if err != nil {
		return nil, fmt.Errorf("failed to list hosts: %w", err)
	}

	return &v1.ListResponse[v1.Host]{
		Kind:       v1.HostKind,
		APIVersion: v1.APIVersion,
		Metadata: v1.ListMeta{
			Continue:           response.Continue,
			RemainingItemCount: response.RemainingItemCount,
		},
		Items: lo.Map(response.Items, func(host *agentmodel.Host, _ int) v1.Host {
			return *mapHostToAPI(host)
		}),
	}, nil
}

// ListAgentsByHost implements usecase.HostManageUsecase.
func (s *Service) ListAgentsByHost(
	ctx context.Context,
	id string,
	options *applicationport.ListOptions,
) (*v1.ListResponse[v1.Agent], error) {
	host, err := s.hostUsecase.GetHost(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get host: %w", err)
	}

	page, err := helper.PaginateUUIDs(host.Status.AgentInstanceUIDs, options.ToDomain())
	if err != nil {
		return nil, fmt.Errorf("failed to paginate host agents: %w", err)
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

			return nil, fmt.Errorf("failed to get agent for host: %w", err)
		}

		items = append(items, *s.mapper.MapAgentToAPI(agent))
	}

	return items, nil
}
