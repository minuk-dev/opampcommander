package usecase

import (
	"context"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

// ServerManageUsecase is a use case that handles server management operations.
type ServerManageUsecase interface {
	ListServers(ctx context.Context) (*v1.ListResponse[v1.Server], error)
}
