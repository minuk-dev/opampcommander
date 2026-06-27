package usecase

import (
	"context"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

// AdminUsecase exposes operational, admin-only views of live OpAMP state.
// It backs the admin API and is meant for operators inspecting the fleet,
// not for agents themselves.
type AdminUsecase interface {
	// ListConnections lists the connections held by the server instance handling
	// the request (node-local view), paged via options.
	ListConnections(ctx context.Context, namespace string,
		options *port.ListOptions) (*v1.ListResponse[v1.Connection], error)
	// ListClusterConnections lists connections across all alive servers,
	// aggregated from the periodic per-server snapshots. Each item carries its
	// owning ServerID.
	ListClusterConnections(ctx context.Context, namespace string,
		options *port.ListOptions) (*v1.ListResponse[v1.Connection], error)
}
