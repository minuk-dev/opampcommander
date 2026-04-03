package client

import (
	"context"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

const (
	// ListRoleURL is the path to list all roles.
	ListRoleURL = "/api/v1/roles"
	// GetRoleURL is the path to get a role by ID.
	GetRoleURL = "/api/v1/roles/{id}"
	// CreateRoleURL is the path to create a new role.
	CreateRoleURL = "/api/v1/roles"
	// UpdateRoleURL is the path to update an existing role.
	UpdateRoleURL = "/api/v1/roles/{id}"
	// DeleteRoleURL is the path to delete a role.
	DeleteRoleURL = "/api/v1/roles/{id}"
)

// RoleService provides methods to interact with roles.
type RoleService struct {
	service *service
}

// NewRoleService creates a new RoleService.
func NewRoleService(service *service) *RoleService {
	return &RoleService{
		service: service,
	}
}

// RoleListResponse represents a list of roles with metadata.
type RoleListResponse = v1.ListResponse[v1.Role]

// ListRoles lists all roles.
func (s *RoleService) ListRoles(
	ctx context.Context,
	opts ...ListOption,
) (*RoleListResponse, error) {
	var listSettings ListSettings
	for _, opt := range opts {
		opt.Apply(&listSettings)
	}

	return listResources[v1.Role](
		ctx,
		s.service,
		ListRoleURL,
		ListSettings{
			limit:          listSettings.limit,
			continueToken:  listSettings.continueToken,
			includeDeleted: listSettings.includeDeleted,
		},
	)
}

// GetRole retrieves a role by its UID.
func (s *RoleService) GetRole(ctx context.Context, uid string) (*v1.Role, error) {
	return getResource[v1.Role](ctx, s.service, GetRoleURL, uid)
}

// CreateRole creates a new role.
func (s *RoleService) CreateRole(ctx context.Context, role *v1.Role) (*v1.Role, error) {
	return createResource[v1.Role, v1.Role](ctx, s.service, CreateRoleURL, role)
}

// UpdateRole updates an existing role.
func (s *RoleService) UpdateRole(ctx context.Context, uid string, role *v1.Role) (*v1.Role, error) {
	return updateResource(ctx, s.service, UpdateRoleURL, uid, role)
}

// DeleteRole deletes a role by its UID.
func (s *RoleService) DeleteRole(ctx context.Context, uid string) error {
	return deleteResource(ctx, s.service, DeleteRoleURL, uid)
}
