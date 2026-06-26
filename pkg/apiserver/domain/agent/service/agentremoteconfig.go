package agentservice

import (
	"context"
	"fmt"
	"time"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var _ agentport.AgentRemoteConfigUsecase = (*AgentRemoteConfigService)(nil)

// AgentRemoteConfigService provides operations for managing agent remote configs,
// including the creation/update lifecycle rules (stamping and immutable-field
// preservation).
type AgentRemoteConfigService struct {
	persistence agentport.AgentRemoteConfigPersistencePort

	// other domain usecases, used to re-run a config's side effects on reconcile.
	endpointDetectionUsecase agentport.EndpointDetectionUsecase
	agentGroupUsecase        agentport.AgentGroupUsecase

	clock clock.Clock
}

// NewAgentRemoteConfigService creates a new AgentRemoteConfigService.
func NewAgentRemoteConfigService(
	persistence agentport.AgentRemoteConfigPersistencePort,
	endpointDetectionUsecase agentport.EndpointDetectionUsecase,
	agentGroupUsecase agentport.AgentGroupUsecase,
) *AgentRemoteConfigService {
	return &AgentRemoteConfigService{
		persistence:              persistence,
		endpointDetectionUsecase: endpointDetectionUsecase,
		agentGroupUsecase:        agentGroupUsecase,
		clock:                    clock.NewRealClock(),
	}
}

// SetClock overrides the clock used for lifecycle timestamps. Intended for tests.
func (s *AgentRemoteConfigService) SetClock(c clock.Clock) {
	s.clock = c
}

// GetAgentRemoteConfig implements [agentport.AgentRemoteConfigUsecase].
func (s *AgentRemoteConfigService) GetAgentRemoteConfig(
	ctx context.Context,
	namespace string,
	name string,
	options *model.GetOptions,
) (*agentmodel.AgentRemoteConfig, error) {
	resource, err := s.persistence.GetAgentRemoteConfig(ctx, namespace, name, options)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent remote config: %w", err)
	}

	// Convert resource to the simpler AgentRemoteConfig type
	return resource, nil
}

// ListAgentRemoteConfigs implements [agentport.AgentRemoteConfigUsecase].
func (s *AgentRemoteConfigService) ListAgentRemoteConfigs(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.AgentRemoteConfig], error) {
	resourceResp, err := s.persistence.ListAgentRemoteConfigs(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list agent remote configs: %w", err)
	}

	return &model.ListResponse[*agentmodel.AgentRemoteConfig]{
		Items:              resourceResp.Items,
		RemainingItemCount: resourceResp.RemainingItemCount,
		Continue:           resourceResp.Continue,
	}, nil
}

// SaveAgentRemoteConfig implements [agentport.AgentRemoteConfigUsecase].
func (s *AgentRemoteConfigService) SaveAgentRemoteConfig(
	ctx context.Context,
	agentremoteconfig *agentmodel.AgentRemoteConfig,
) (*agentmodel.AgentRemoteConfig, error) {
	resource, err := s.persistence.PutAgentRemoteConfig(ctx, agentremoteconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to save agent remote config: %w", err)
	}

	// Convert resource to the simpler AgentRemoteConfig type
	return resource, nil
}

// CreateAgentRemoteConfig implements [agentport.AgentRemoteConfigUsecase].
func (s *AgentRemoteConfigService) CreateAgentRemoteConfig(
	ctx context.Context,
	agentRemoteConfig *agentmodel.AgentRemoteConfig,
	actor string,
) (*agentmodel.AgentRemoteConfig, error) {
	agentRemoteConfig.MarkAsCreated(s.clock.Now(), actor)

	created, err := s.persistence.PutAgentRemoteConfig(ctx, agentRemoteConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent remote config: %w", err)
	}

	return created, nil
}

// UpdateAgentRemoteConfig implements [agentport.AgentRemoteConfigUsecase].
func (s *AgentRemoteConfigService) UpdateAgentRemoteConfig(
	ctx context.Context,
	namespace string,
	name string,
	agentRemoteConfig *agentmodel.AgentRemoteConfig,
) (*agentmodel.AgentRemoteConfig, error) {
	existing, err := s.persistence.GetAgentRemoteConfig(ctx, namespace, name, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent remote config for update: %w", err)
	}

	existing.ApplyUpdate(agentRemoteConfig)

	updated, err := s.persistence.PutAgentRemoteConfig(ctx, existing)
	if err != nil {
		return nil, fmt.Errorf("failed to update agent remote config: %w", err)
	}

	return updated, nil
}

// DeleteAgentRemoteConfig implements [agentport.AgentRemoteConfigUsecase].
func (s *AgentRemoteConfigService) DeleteAgentRemoteConfig(
	ctx context.Context,
	namespace string,
	name string,
	deletedAt time.Time,
	deletedBy string,
) error {
	resource, err := s.persistence.GetAgentRemoteConfig(ctx, namespace, name, nil)
	if err != nil {
		return fmt.Errorf("failed to get agent remote config for deletion: %w", err)
	}

	resource.MarkDeleted(deletedAt, deletedBy)

	_, err = s.persistence.PutAgentRemoteConfig(ctx, resource)
	if err != nil {
		return fmt.Errorf("failed to mark agent remote config as deleted: %w", err)
	}

	return nil
}

// ReconcileAgentRemoteConfig implements [agentport.AgentRemoteConfigUsecase]. It loads the
// named config and re-runs the side effects that normally fire on create/update: telemetry
// endpoint detection from the config's collector exporters, then re-propagation of the config
// to every agent group that references it. Both run synchronously so the caller learns of
// failures (unlike the fire-and-forget triggers on the write path).
func (s *AgentRemoteConfigService) ReconcileAgentRemoteConfig(
	ctx context.Context,
	namespace string,
	name string,
) error {
	remoteConfig, err := s.persistence.GetAgentRemoteConfig(ctx, namespace, name, nil)
	if err != nil {
		return fmt.Errorf("get agent remote config %s/%s: %w", namespace, name, err)
	}

	err = s.endpointDetectionUsecase.ReconcileEndpointsFromRemoteConfig(ctx, remoteConfig)
	if err != nil {
		return fmt.Errorf("reconcile endpoints from remote config %s/%s: %w", namespace, name, err)
	}

	err = s.agentGroupUsecase.PropagateAgentRemoteConfigChange(ctx, namespace, name)
	if err != nil {
		return fmt.Errorf("propagate remote config change %s/%s: %w", namespace, name, err)
	}

	return nil
}
