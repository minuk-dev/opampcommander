package usecase

import (
	"context"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

// AgentPackageManageUsecase is a use case that handles agent package operations.
type AgentPackageManageUsecase interface {
	GetAgentPackage(ctx context.Context, namespace string, name string,
		options *port.GetOptions) (*v1.AgentPackage, error)
	ListAgentPackages(ctx context.Context, options *port.ListOptions) (*v1.ListResponse[v1.AgentPackage], error)
	CreateAgentPackage(ctx context.Context, agentPackage *v1.AgentPackage) (*v1.AgentPackage, error)
	UpdateAgentPackage(ctx context.Context, namespace string, name string,
		agentPackage *v1.AgentPackage) (*v1.AgentPackage, error)
	DeleteAgentPackage(ctx context.Context, namespace string, name string) error
}
