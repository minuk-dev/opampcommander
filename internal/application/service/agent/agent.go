// Package agent provides application services for the agent
package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"k8s.io/utils/clock"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/internal/application/helper"
	applicationport "github.com/minuk-dev/opampcommander/internal/application/port"
	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/internal/domain/agent/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

var (
	// ErrRestartCapabilityNotSupported is returned when agent doesn't support restart capability.
	ErrRestartCapabilityNotSupported = errors.New("agent does not support restart capability")
	// ErrAgentNamespaceMismatch is returned when the agent does not belong to the specified namespace.
	ErrAgentNamespaceMismatch = errors.New("agent does not belong to the specified namespace")
)

var _ applicationport.AgentManageUsecase = (*Service)(nil)

// Service is a struct that implements the AgentManageUsecase interface.
type Service struct {
	// domain usecases
	agentUsecase             agentport.AgentUsecase
	agentNotificationUsecase agentport.AgentNotificationUsecase

	// mapper
	mapper *helper.Mapper
	logger *slog.Logger
	clock  clock.Clock
}

// New creates a new instance of the Service struct.
func New(
	agentUsecase agentport.AgentUsecase,
	agentNotificationUsecase agentport.AgentNotificationUsecase,
	logger *slog.Logger,
) *Service {
	return &Service{
		agentUsecase:             agentUsecase,
		agentNotificationUsecase: agentNotificationUsecase,

		mapper: helper.NewMapper(),
		logger: logger,
		clock:  clock.RealClock{},
	}
}

// GetAgent implements port.AgentManageUsecase.
func (s *Service) GetAgent(
	ctx context.Context,
	namespace string,
	instanceUID uuid.UUID,
) (*v1.Agent, error) {
	agent, err := s.agentUsecase.GetAgent(ctx, instanceUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	if agent.Metadata.Namespace != namespace {
		return nil, fmt.Errorf("failed to get agent: %w", ErrAgentNamespaceMismatch)
	}

	return s.mapper.MapAgentToAPI(agent), nil
}

// ListAgents implements port.AgentManageUsecase.
func (s *Service) ListAgents(
	ctx context.Context,
	namespace string,
	options *model.ListOptions,
) (*v1.ListResponse[v1.Agent], error) {
	response, err := s.agentUsecase.ListAgents(ctx, namespace, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}

	return &v1.ListResponse[v1.Agent]{
		Kind:       v1.AgentKind,
		APIVersion: v1.APIVersion,
		Metadata: v1.ListMeta{
			Continue:           response.Continue,
			RemainingItemCount: response.RemainingItemCount,
		},
		Items: lo.Map(response.Items, func(agent *agentmodel.Agent, _ int) v1.Agent {
			return *s.mapper.MapAgentToAPI(agent)
		}),
	}, nil
}

// SearchAgents implements port.AgentManageUsecase.
func (s *Service) SearchAgents(
	ctx context.Context,
	namespace string,
	query string,
	options *model.ListOptions,
) (*v1.ListResponse[v1.Agent], error) {
	response, err := s.agentUsecase.SearchAgents(ctx, namespace, query, options)
	if err != nil {
		return nil, fmt.Errorf("failed to search agents: %w", err)
	}

	return &v1.ListResponse[v1.Agent]{
		Kind:       v1.AgentKind,
		APIVersion: v1.APIVersion,
		Metadata: v1.ListMeta{
			Continue:           response.Continue,
			RemainingItemCount: response.RemainingItemCount,
		},
		Items: lo.Map(response.Items, func(agent *agentmodel.Agent, _ int) v1.Agent {
			return *s.mapper.MapAgentToAPI(agent)
		}),
	}, nil
}

// UpdateAgent implements [port.AgentManageUsecase].
func (s *Service) UpdateAgent(
	ctx context.Context,
	namespace string,
	instanceUID uuid.UUID,
	api *v1.Agent,
) (*v1.Agent, error) {
	existing, err := s.agentUsecase.GetAgent(ctx, instanceUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	if existing.Metadata.Namespace != namespace {
		return nil, fmt.Errorf("failed to update agent: %w", ErrAgentNamespaceMismatch)
	}

	agent := s.mapper.MapAPIToAgent(api)

	// Handle restart request
	if !agent.Spec.RestartInfo.RequiredRestartedAt.IsZero() {
		restartErr := existing.SetRestartRequired(agent.Spec.RestartInfo.RequiredRestartedAt)
		if restartErr != nil {
			return nil, fmt.Errorf("failed to set restart required: %w", restartErr)
		}
	}

	// Update other spec fields if provided
	if agent.Spec.NewInstanceUID != uuid.Nil {
		existing.Spec.NewInstanceUID = agent.Spec.NewInstanceUID
	}

	if agent.Spec.RemoteConfig != nil && len(agent.Spec.RemoteConfig.ConfigMap.ConfigMap) > 0 {
		existing.Spec.RemoteConfig = agent.Spec.RemoteConfig
	}

	err = s.agentUsecase.SaveAgent(ctx, existing)
	if err != nil {
		return nil, fmt.Errorf("failed to update agent: %w", err)
	}

	// Notify about agent update
	notifyErr := s.agentNotificationUsecase.NotifyAgentUpdated(ctx, existing)
	if notifyErr != nil {
		s.logger.Error("failed to notify agent updated", "error", notifyErr.Error())
	}

	return s.mapper.MapAgentToAPI(existing), nil
}
