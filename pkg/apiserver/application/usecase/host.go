package usecase

import (
	"context"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

// HostManageUsecase exposes the host aggregates auto-discovered from agents'
// reported attributes (one per host/VM an agent runs on). It is read-only
// and backs the /api/v1/hosts controller.
type HostManageUsecase interface {
	// GetHost returns the host aggregate with the given id, or
	// model.ErrResourceNotExist if none was discovered.
	GetHost(ctx context.Context, id string) (*v1.Host, error)
	// ListHosts returns a paged list of discovered hosts.
	ListHosts(ctx context.Context, options *port.ListOptions) (*v1.ListResponse[v1.Host], error)
	// ListAgentsByHost returns the agents running on the host.
	ListAgentsByHost(ctx context.Context, id string,
		options *port.ListOptions) (*v1.ListResponse[v1.Agent], error)
}
