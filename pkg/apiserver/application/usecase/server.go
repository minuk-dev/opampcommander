package usecase

import (
	"context"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

// ServerManageUsecase exposes the opampcommander server instances taking
// part in multi-server coordination. It is read-only and backs the
// /api/v1/servers controller.
type ServerManageUsecase interface {
	// ListServers returns every opampcommander server instance known to the
	// coordination layer.
	ListServers(ctx context.Context) (*v1.ListResponse[v1.Server], error)
}
