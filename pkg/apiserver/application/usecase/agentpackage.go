package usecase

import (
	"context"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

// AgentPackageManageUsecase manages agent packages: downloadable artifacts
// (collector binaries, plugins) the server can offer to agents over OpAMP.
// It backs the /api/v1/agentpackages controller.
type AgentPackageManageUsecase interface {
	// GetAgentPackage returns the named package in namespace, or
	// model.ErrResourceNotExist if absent.
	GetAgentPackage(ctx context.Context, namespace string, name string,
		options *port.GetOptions) (*v1.AgentPackage, error)
	// ListAgentPackages returns a paged list of packages across namespaces.
	ListAgentPackages(ctx context.Context, options *port.ListOptions) (*v1.ListResponse[v1.AgentPackage], error)
	// CreateAgentPackage persists a new package, returning
	// model.ErrResourceAlreadyExist on a duplicate.
	CreateAgentPackage(ctx context.Context, agentPackage *v1.AgentPackage) (*v1.AgentPackage, error)
	// UpdateAgentPackage replaces the named package; optimistic-concurrency
	// controlled (model.ErrConflict on a stale write).
	UpdateAgentPackage(ctx context.Context, namespace string, name string,
		agentPackage *v1.AgentPackage) (*v1.AgentPackage, error)
	// DeleteAgentPackage removes the named package.
	DeleteAgentPackage(ctx context.Context, namespace string, name string) error
}
