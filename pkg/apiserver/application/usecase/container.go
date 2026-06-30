package usecase

import (
	"context"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

// ContainerManageUsecase exposes the container aggregates auto-discovered
// from agents' reported attributes (one per container an agent runs in).
// It is read-only and backs the /api/v1/containers controller.
type ContainerManageUsecase interface {
	// GetContainer returns the container aggregate with the given id, or
	// model.ErrResourceNotExist if none was discovered.
	GetContainer(ctx context.Context, id string) (*v1.Container, error)
	// ListContainers returns a paged list of discovered containers.
	ListContainers(ctx context.Context, options *port.ListOptions) (*v1.ListResponse[v1.Container], error)
	// ListAgentsByContainer returns the agents running in the container.
	ListAgentsByContainer(ctx context.Context, id string,
		options *port.ListOptions) (*v1.ListResponse[v1.Agent], error)
}
