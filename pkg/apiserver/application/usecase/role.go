package usecase

import (
	"context"

	"github.com/google/uuid"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

// RoleManageUsecase manages RBAC roles (named permission sets), keyed by
// UUID. It backs the /api/v1/roles controller.
type RoleManageUsecase interface {
	// GetRole returns the role with the given UID, or model.ErrResourceNotExist if
	// absent.
	GetRole(ctx context.Context, uid uuid.UUID, options *port.GetOptions) (*v1.Role, error)
	// ListRoles returns a paged list of roles.
	ListRoles(ctx context.Context, options *port.ListOptions) (*v1.ListResponse[v1.Role], error)
	// CreateRole persists a new role.
	CreateRole(ctx context.Context, role *v1.Role) (*v1.Role, error)
	// UpdateRole replaces the role with the given UID.
	UpdateRole(ctx context.Context, uid uuid.UUID, role *v1.Role) (*v1.Role, error)
	// DeleteRole removes the role with the given UID.
	DeleteRole(ctx context.Context, uid uuid.UUID) error
}
