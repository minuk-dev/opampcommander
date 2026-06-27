package usecase

import (
	"context"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

// AgentRemoteConfigManageUsecase manages agent remote configurations: the
// desired OpAMP config documents the server pushes to agents. It backs the
// /api/v1/agentremoteconfigs controller.
type AgentRemoteConfigManageUsecase interface {
	// GetAgentRemoteConfig returns the named remote config in namespace, or
	// model.ErrResourceNotExist if absent.
	GetAgentRemoteConfig(ctx context.Context, namespace string,
		name string, options *port.GetOptions) (*v1.AgentRemoteConfig, error)
	// ListAgentRemoteConfigs returns a paged list across namespaces.
	ListAgentRemoteConfigs(ctx context.Context,
		options *port.ListOptions) (*v1.ListResponse[v1.AgentRemoteConfig], error)
	// CreateAgentRemoteConfig persists a new remote config, returning
	// model.ErrResourceAlreadyExist on a duplicate.
	CreateAgentRemoteConfig(ctx context.Context,
		agentRemoteConfig *v1.AgentRemoteConfig) (*v1.AgentRemoteConfig, error)
	// UpdateAgentRemoteConfig replaces the named remote config;
	// optimistic-concurrency controlled (model.ErrConflict on a stale write).
	UpdateAgentRemoteConfig(ctx context.Context, namespace string, name string,
		agentRemoteConfig *v1.AgentRemoteConfig) (*v1.AgentRemoteConfig, error)
	// DeleteAgentRemoteConfig removes the named remote config.
	DeleteAgentRemoteConfig(ctx context.Context, namespace string, name string) error
}
