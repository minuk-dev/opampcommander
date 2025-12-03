// Package agent provides application services for the agent
package agent

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/samber/lo"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	v1agent "github.com/minuk-dev/opampcommander/api/v1/agent"
	"github.com/minuk-dev/opampcommander/internal/application/mapper"
	applicationport "github.com/minuk-dev/opampcommander/internal/application/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ applicationport.AgentManageUsecase = (*Service)(nil)

// Service is a struct that implements the AgentManageUsecase interface.
type Service struct {
	// domain usecases
	agentUsecase             domainport.AgentUsecase
	agentNotificationUsecase domainport.AgentNotificationUsecase

	// mapper
	mapper *mapper.Mapper
	logger *slog.Logger
}

// New creates a new instance of the Service struct.
func New(
	agentUsecase domainport.AgentUsecase,
	agentNotificationUsecase domainport.AgentNotificationUsecase,
	logger *slog.Logger,
) *Service {
	return &Service{
		agentUsecase:             agentUsecase,
		agentNotificationUsecase: agentNotificationUsecase,

		mapper: mapper.New(),
		logger: logger,
	}
}

// GetAgent implements port.AgentManageUsecase.
func (s *Service) GetAgent(ctx context.Context, instanceUID uuid.UUID) (*v1agent.Agent, error) {
	agent, err := s.agentUsecase.GetAgent(ctx, instanceUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	return s.mapper.MapAgentToAPI(agent), nil
}

// ListAgents implements port.AgentManageUsecase.
func (s *Service) ListAgents(
	ctx context.Context,
	options *model.ListOptions,
) (*v1agent.ListResponse, error) {
	response, err := s.agentUsecase.ListAgents(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}

	return v1agent.NewListResponse(
		lo.Map(response.Items, func(agent *model.Agent, _ int) v1agent.Agent {
			return *s.mapper.MapAgentToAPI(agent)
		}),
		v1.ListMeta{
			Continue:           response.Continue,
			RemainingItemCount: response.RemainingItemCount,
		},
	), nil
}

// SetNewInstanceUID implements port.AgentManageUsecase.
func (s *Service) SetNewInstanceUID(
	ctx context.Context,
	instanceUID uuid.UUID,
	newInstanceUID string,
) (*v1agent.Agent, error) {
	agent, err := s.agentUsecase.GetAgent(ctx, instanceUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	// Set the new instance UID
	agent.Spec.NewInstanceUID = []byte(newInstanceUID)

	// Save the updated agent
	err = s.agentUsecase.SaveAgent(ctx, agent)
	if err != nil {
		return nil, fmt.Errorf("failed to save agent: %w", err)
	}

	return s.mapper.MapAgentToAPI(agent), nil
}
