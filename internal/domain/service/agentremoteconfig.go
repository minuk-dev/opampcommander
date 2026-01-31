package service

import (
	"context"
	"fmt"
	"time"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ port.AgentRemoteConfigUsecase = (*AgentRemoteConfigService)(nil)

// AgentRemoteConfigService provides operations for managing agent remote configs.
type AgentRemoteConfigService struct {
	persistence port.AgentRemoteConfigPersistencePort
}

// NewAgentRemoteConfigService creates a new AgentRemoteConfigService.
func NewAgentRemoteConfigService(persistence port.AgentRemoteConfigPersistencePort) *AgentRemoteConfigService {
	return &AgentRemoteConfigService{
		persistence: persistence,
	}
}

// GetAgentRemoteConfig implements [port.AgentRemoteConfigUsecase].
func (s *AgentRemoteConfigService) GetAgentRemoteConfig(
	ctx context.Context,
	name string,
) (*model.AgentRemoteConfig, error) {
	resource, err := s.persistence.GetAgentRemoteConfig(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent remote config: %w", err)
	}

	// Convert resource to the simpler AgentRemoteConfig type
	return resource, nil
}

// ListAgentRemoteConfigs implements [port.AgentRemoteConfigUsecase].
func (s *AgentRemoteConfigService) ListAgentRemoteConfigs(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*model.AgentRemoteConfig], error) {
	resourceResp, err := s.persistence.ListAgentRemoteConfigs(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list agent remote configs: %w", err)
	}

	return &model.ListResponse[*model.AgentRemoteConfig]{
		Items:              resourceResp.Items,
		RemainingItemCount: resourceResp.RemainingItemCount,
		Continue:           resourceResp.Continue,
	}, nil
}

// SaveAgentRemoteConfig implements [port.AgentRemoteConfigUsecase].
func (s *AgentRemoteConfigService) SaveAgentRemoteConfig(
	ctx context.Context,
	agentremoteconfig *model.AgentRemoteConfig,
) (*model.AgentRemoteConfig, error) {
	resource, err := s.persistence.PutAgentRemoteConfig(ctx, agentremoteconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to save agent remote config: %w", err)
	}

	// Convert resource to the simpler AgentRemoteConfig type
	return resource, nil
}

// DeleteAgentRemoteConfig implements [port.AgentRemoteConfigUsecase].
func (s *AgentRemoteConfigService) DeleteAgentRemoteConfig(
	ctx context.Context,
	name string,
	deletedAt time.Time,
	deletedBy string,
) error {
	resource, err := s.persistence.GetAgentRemoteConfig(ctx, name)
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
