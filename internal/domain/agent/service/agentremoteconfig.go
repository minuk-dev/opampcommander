package agentservice

import (
	"context"
	"fmt"
	"time"

	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/internal/domain/agent/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

var _ agentport.AgentRemoteConfigUsecase = (*AgentRemoteConfigService)(nil)

// AgentRemoteConfigService provides operations for managing agent remote configs.
type AgentRemoteConfigService struct {
	persistence agentport.AgentRemoteConfigPersistencePort
}

// NewAgentRemoteConfigService creates a new AgentRemoteConfigService.
func NewAgentRemoteConfigService(persistence agentport.AgentRemoteConfigPersistencePort) *AgentRemoteConfigService {
	return &AgentRemoteConfigService{
		persistence: persistence,
	}
}

// GetAgentRemoteConfig implements [agentport.AgentRemoteConfigUsecase].
func (s *AgentRemoteConfigService) GetAgentRemoteConfig(
	ctx context.Context,
	namespace string,
	name string,
) (*agentmodel.AgentRemoteConfig, error) {
	resource, err := s.persistence.GetAgentRemoteConfig(ctx, namespace, name)
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

// DeleteAgentRemoteConfig implements [agentport.AgentRemoteConfigUsecase].
func (s *AgentRemoteConfigService) DeleteAgentRemoteConfig(
	ctx context.Context,
	namespace string,
	name string,
	deletedAt time.Time,
	deletedBy string,
) error {
	resource, err := s.persistence.GetAgentRemoteConfig(ctx, namespace, name)
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
