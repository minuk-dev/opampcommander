package usecase

import (
	"context"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

// EndpointManageUsecase manages endpoints: telemetry-backend destinations
// (with their headers and tags) that agents export telemetry to. It backs
// the /api/v1/endpoints controller.
type EndpointManageUsecase interface {
	// GetEndpoint returns the named endpoint in namespace, or
	// model.ErrResourceNotExist if absent.
	GetEndpoint(ctx context.Context, namespace string,
		name string, options *port.GetOptions) (*v1.Endpoint, error)
	// ListEndpoints returns a paged list of endpoints in namespace.
	ListEndpoints(ctx context.Context, namespace string,
		options *port.ListOptions) (*v1.ListResponse[v1.Endpoint], error)
	// CreateEndpoint persists a new endpoint, returning
	// model.ErrResourceAlreadyExist on a duplicate.
	CreateEndpoint(ctx context.Context,
		endpoint *v1.Endpoint) (*v1.Endpoint, error)
	// UpdateEndpoint replaces the named endpoint; optimistic-concurrency
	// controlled (model.ErrConflict on a stale write).
	UpdateEndpoint(ctx context.Context, namespace string, name string,
		endpoint *v1.Endpoint) (*v1.Endpoint, error)
	// DeleteEndpoint removes the named endpoint.
	DeleteEndpoint(ctx context.Context, namespace string, name string) error
}
