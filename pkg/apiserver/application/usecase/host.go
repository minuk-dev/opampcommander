package usecase

import (
	"context"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

// HostManageUsecase is a use case that handles host management operations.
type HostManageUsecase interface {
	GetHost(ctx context.Context, id string) (*v1.Host, error)
	ListHosts(ctx context.Context, options *port.ListOptions) (*v1.ListResponse[v1.Host], error)
	ListAgentsByHost(ctx context.Context, id string,
		options *port.ListOptions) (*v1.ListResponse[v1.Agent], error)
}
