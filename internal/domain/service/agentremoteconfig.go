package service

import (
	"context"
	"time"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ port.AgentRemoteConfigUsecase = (*AgentRemoteConfigService)(nil)

type AgentRemoteConfigService struct {
}

// GetAgentRemoteConfig implements [port.AgentRemoteConfigUsecase].
func (a *AgentRemoteConfigService) GetAgentRemoteConfig(ctx context.Context, name string) (*model.AgentRemoteConfig, error) {
	panic("unimplemented")
}

// ListAgentRemoteConfigs implements [port.AgentRemoteConfigUsecase].
func (a *AgentRemoteConfigService) ListAgentRemoteConfigs(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.AgentRemoteConfig], error) {
	panic("unimplemented")
}

// SaveAgentRemoteConfig implements [port.AgentRemoteConfigUsecase].
func (a *AgentRemoteConfigService) SaveAgentRemoteConfig(ctx context.Context, agentRemoteConfig *model.AgentRemoteConfig) (*model.AgentRemoteConfig, error) {
	panic("unimplemented")
}

// DeleteAgentRemoteConfig implements [port.AgentRemoteConfigUsecase].
func (a *AgentRemoteConfigService) DeleteAgentRemoteConfig(ctx context.Context, name string, deletedAt time.Time, deletedBy string) error {
	panic("unimplemented")
}
