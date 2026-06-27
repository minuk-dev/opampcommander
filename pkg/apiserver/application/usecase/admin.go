package usecase

import (
	"context"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

// AdminUsecase is a use case that handles admin operations.
type AdminUsecase interface {
	ListConnections(ctx context.Context, namespace string,
		options *port.ListOptions) (*v1.ListResponse[v1.Connection], error)
	ListClusterConnections(ctx context.Context, namespace string,
		options *port.ListOptions) (*v1.ListResponse[v1.Connection], error)
}
