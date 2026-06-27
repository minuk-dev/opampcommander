package usecase

import (
	"context"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

// ContainerManageUsecase is a use case that handles container management operations.
type ContainerManageUsecase interface {
	GetContainer(ctx context.Context, id string) (*v1.Container, error)
	ListContainers(ctx context.Context, options *port.ListOptions) (*v1.ListResponse[v1.Container], error)
	ListAgentsByContainer(ctx context.Context, id string,
		options *port.ListOptions) (*v1.ListResponse[v1.Agent], error)
}
