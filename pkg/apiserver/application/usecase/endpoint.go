package usecase

import (
	"context"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

// EndpointManageUsecase is a use case that handles endpoint management operations.
type EndpointManageUsecase interface {
	GetEndpoint(ctx context.Context, namespace string,
		name string, options *port.GetOptions) (*v1.Endpoint, error)
	ListEndpoints(ctx context.Context, namespace string,
		options *port.ListOptions) (*v1.ListResponse[v1.Endpoint], error)
	CreateEndpoint(ctx context.Context,
		endpoint *v1.Endpoint) (*v1.Endpoint, error)
	UpdateEndpoint(ctx context.Context, namespace string, name string,
		endpoint *v1.Endpoint) (*v1.Endpoint, error)
	DeleteEndpoint(ctx context.Context, namespace string, name string) error
}
