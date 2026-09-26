package usecase

import (
	"context"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

// ApplicationManageUsecase exposes logical services auto-discovered from an
// agent's service.* attributes.
// It is read-only and backs the /api/v1/applications controller.
type ApplicationManageUsecase interface {
	// GetApplication returns the application aggregate with the given id, or
	// model.ErrResourceNotExist if none was discovered.
	GetApplication(ctx context.Context, id string) (*v1.Application, error)
	// ListApplications returns a paged list of discovered applications.
	ListApplications(ctx context.Context, options *port.ListOptions) (*v1.ListResponse[v1.Application], error)
	// ListAgentsByApplication returns the agents running in the application.
	ListAgentsByApplication(ctx context.Context, id string,
		options *port.ListOptions) (*v1.ListResponse[v1.Agent], error)
}
