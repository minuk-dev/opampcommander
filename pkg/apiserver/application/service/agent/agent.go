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
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/helper"
	applicationport "github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/usecase"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
)

var (
	// ErrRestartCapabilityNotSupported is returned when agent doesn't support restart capability.
	ErrRestartCapabilityNotSupported = errors.New("agent does not support restart capability")
	// ErrAgentNamespaceMismatch is returned when the agent does not belong to the specified namespace.
	// It aliases the application port sentinel so the HTTP layer can map it to a 404 while existing
	// references to this package-level name keep working.
	ErrAgentNamespaceMismatch = applicationport.ErrAgentNamespaceMismatch
)

var _ usecase.AgentManageUsecase = (*Service)(nil)

// Service is a struct that implements the AgentManageUsecase interface.
type Service struct {
	// domain usecases
	agentUsecase               agentport.AgentUsecase
	agentNotificationUsecase   agentport.AgentNotificationUsecase
	endpointDetectionUsecase   agentport.EndpointDetectionUsecase
	cacheInvalidationPublisher agentport.AgentCacheInvalidationPublisher

	// mapper
	mapper *helper.Mapper
	logger *slog.Logger
}

// New creates a new instance of the Service struct.
func New(
	agentUsecase agentport.AgentUsecase,
	agentNotificationUsecase agentport.AgentNotificationUsecase,
	endpointDetectionUsecase agentport.EndpointDetectionUsecase,
	cacheInvalidationPublisher agentport.AgentCacheInvalidationPublisher,
	logger *slog.Logger,
) *Service {
	realClock := clock.RealClock{}

	return &Service{
		agentUsecase:               agentUsecase,
		agentNotificationUsecase:   agentNotificationUsecase,
		endpointDetectionUsecase:   endpointDetectionUsecase,
		cacheInvalidationPublisher: cacheInvalidationPublisher,

		mapper: helper.NewMapper(realClock, agentmodel.DefaultConnectionStaleness),
		logger: logger,
	}
}

// ListAgentEndpoints implements usecase.AgentManageUsecase. It returns a read-only view
// of the endpoints the agent currently exports to, extracted from its reported
// effective configuration (not persisted Endpoint resources).
func (s *Service) ListAgentEndpoints(
	ctx context.Context,
	namespace string,
	instanceUID uuid.UUID,
) (*v1.ListResponse[v1.Endpoint], error) {
	agent, err := s.getAgentInNamespace(ctx, namespace, instanceUID)
	if err != nil {
		return nil, err
	}

	endpoints, err := s.endpointDetectionUsecase.ExtractEndpointsFromAgent(agent)
	if err != nil {
		return nil, fmt.Errorf("extract endpoints from agent effective config: %w", err)
	}

	return &v1.ListResponse[v1.Endpoint]{
		Kind:       v1.EndpointKind,
		APIVersion: v1.APIVersion,
		Metadata:   v1.ListMeta{Continue: "", RemainingItemCount: 0},
		Items: lo.Map(endpoints, func(item *agentmodel.Endpoint, _ int) v1.Endpoint {
			return *s.mapper.MapEndpointToAPI(item)
		}),
	}, nil
}

// GetAgent implements usecase.AgentManageUsecase.
func (s *Service) GetAgent(
	ctx context.Context,
	namespace string,
	instanceUID uuid.UUID,
) (*v1.Agent, error) {
	agent, err := s.getAgentInNamespace(ctx, namespace, instanceUID)
	if err != nil {
		return nil, err
	}

	return s.mapper.MapAgentToAPI(agent), nil
}

// ListAgents implements usecase.AgentManageUsecase.
func (s *Service) ListAgents(
	ctx context.Context,
	namespace string,
	options *applicationport.ListOptions,
) (*v1.ListResponse[v1.Agent], error) {
	response, err := s.agentUsecase.ListAgents(ctx, namespace, options.ToDomain())
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

// SearchAgents implements usecase.AgentManageUsecase.
func (s *Service) SearchAgents(
	ctx context.Context,
	namespace string,
	query string,
	options *applicationport.ListOptions,
) (*v1.ListResponse[v1.Agent], error) {
	response, err := s.agentUsecase.SearchAgents(ctx, namespace, query, options.ToDomain())
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

// DeleteAgent implements [usecase.AgentManageUsecase].
//
// Only disconnected agents may be deleted. The connection guard is enforced by the
// domain DeleteAgent (against a fresh read), so a still-connected agent is rejected
// with [applicationport.ErrAgentConnected]; this method just scopes the delete to the
// requested namespace.
func (s *Service) DeleteAgent(
	ctx context.Context,
	namespace string,
	instanceUID uuid.UUID,
) error {
	_, err := s.getAgentInNamespace(ctx, namespace, instanceUID)
	if err != nil {
		return err
	}

	err = s.agentUsecase.DeleteAgent(ctx, instanceUID)
	if err != nil {
		return fmt.Errorf("failed to delete agent: %w", err)
	}

	s.invalidatePeerCaches(ctx, instanceUID)

	return nil
}

// UpdateAgent implements [usecase.AgentManageUsecase].
func (s *Service) UpdateAgent(
	ctx context.Context,
	namespace string,
	instanceUID uuid.UUID,
	api *v1.Agent,
) (*v1.Agent, error) {
	existing, err := s.getAgentInNamespace(ctx, namespace, instanceUID)
	if err != nil {
		return nil, err
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

	s.invalidatePeerCaches(ctx, instanceUID)

	return s.mapper.MapAgentToAPI(existing), nil
}

// invalidatePeerCaches asks other nodes to drop their cached copy of the agent after a
// local API mutation, so they don't serve it stale until their TTL expires. It is
// best-effort: failures are logged, never surfaced to the API caller, since the entry
// expires on its own and the local node's cache was already updated by the write.
func (s *Service) invalidatePeerCaches(ctx context.Context, instanceUID uuid.UUID) {
	err := s.cacheInvalidationPublisher.BroadcastAgentCacheInvalidation(ctx, instanceUID)
	if err != nil {
		s.logger.Error("failed to broadcast agent cache invalidation",
			"instanceUID", instanceUID.String(), "error", err.Error())
	}
}

// getAgentInNamespace fetches an agent by UID and verifies it belongs to the given
// namespace. It returns ErrAgentNamespaceMismatch when the agent exists but in a
// different namespace, so the HTTP layer can map that to a 404. Centralising this
// keeps the namespace-scoping invariant in one place for Get/Update/Delete.
func (s *Service) getAgentInNamespace(
	ctx context.Context,
	namespace string,
	instanceUID uuid.UUID,
) (*agentmodel.Agent, error) {
	agent, err := s.agentUsecase.GetAgent(ctx, instanceUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	if agent.Metadata.Namespace != namespace {
		return nil, fmt.Errorf("failed to get agent: %w", ErrAgentNamespaceMismatch)
	}

	return agent, nil
}
