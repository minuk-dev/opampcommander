package usecase

import (
	"context"

	"github.com/google/uuid"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

// RoleManageUsecase is a use case that handles role management operations.
type RoleManageUsecase interface {
	GetRole(ctx context.Context, uid uuid.UUID, options *port.GetOptions) (*v1.Role, error)
	ListRoles(ctx context.Context, options *port.ListOptions) (*v1.ListResponse[v1.Role], error)
	CreateRole(ctx context.Context, role *v1.Role) (*v1.Role, error)
	UpdateRole(ctx context.Context, uid uuid.UUID, role *v1.Role) (*v1.Role, error)
	DeleteRole(ctx context.Context, uid uuid.UUID) error
}
