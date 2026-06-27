package usecase

import (
	"context"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

// AgentRemoteConfigManageUsecase is a use case that handles agent remote config management operations.
type AgentRemoteConfigManageUsecase interface {
	GetAgentRemoteConfig(ctx context.Context, namespace string,
		name string, options *port.GetOptions) (*v1.AgentRemoteConfig, error)
	ListAgentRemoteConfigs(ctx context.Context,
		options *port.ListOptions) (*v1.ListResponse[v1.AgentRemoteConfig], error)
	CreateAgentRemoteConfig(ctx context.Context,
		agentRemoteConfig *v1.AgentRemoteConfig) (*v1.AgentRemoteConfig, error)
	UpdateAgentRemoteConfig(ctx context.Context, namespace string, name string,
		agentRemoteConfig *v1.AgentRemoteConfig) (*v1.AgentRemoteConfig, error)
	DeleteAgentRemoteConfig(ctx context.Context, namespace string, name string) error
}
