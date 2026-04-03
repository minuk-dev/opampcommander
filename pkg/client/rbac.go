package client

import (
	"context"
	"fmt"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

const (
	// AssignRoleURL is the path to assign a role to a user.
	AssignRoleURL = "/api/v1/rbac/assign"
	// UnassignRoleURL is the path to unassign a role from a user.
	UnassignRoleURL = "/api/v1/rbac/unassign"
	// CheckPermissionURL is the path to check a user's permission.
	CheckPermissionURL = "/api/v1/rbac/check"
	// GetUserRolesURL is the path to get a user's roles.
	GetUserRolesURL = "/api/v1/rbac/users/{id}/roles"
	// GetUserPermissionsURL is the path to get a user's permissions.
	GetUserPermissionsURL = "/api/v1/rbac/users/{id}/permissions"
	// SyncPoliciesURL is the path to sync RBAC policies.
	SyncPoliciesURL = "/api/v1/rbac/sync"
)

// RBACService provides methods to interact with RBAC.
type RBACService struct {
	service *service
}

// NewRBACService creates a new RBACService.
func NewRBACService(service *service) *RBACService {
	return &RBACService{
		service: service,
	}
}

// AssignRole assigns a role to a user.
func (s *RBACService) AssignRole(ctx context.Context, req *v1.AssignRoleRequest) error {
	res, err := s.service.Resty.R().
		SetContext(ctx).
		SetBody(req).
		Post(AssignRoleURL)
	if err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}

	if res.IsError() {
		return fmt.Errorf("failed to assign role: %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	return nil
}

// UnassignRole unassigns a role from a user.
func (s *RBACService) UnassignRole(ctx context.Context, userID, roleID string) error {
	req := struct {
		UserID string `json:"userId"`
		RoleID string `json:"roleId"`
	}{
		UserID: userID,
		RoleID: roleID,
	}

	res, err := s.service.Resty.R().
		SetContext(ctx).
		SetBody(&req).
		Post(UnassignRoleURL)
	if err != nil {
		return fmt.Errorf("failed to unassign role: %w", err)
	}

	if res.IsError() {
		return fmt.Errorf("failed to unassign role: %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	return nil
}

// CheckPermission checks whether a user has a specific permission.
func (s *RBACService) CheckPermission(
	ctx context.Context,
	req *v1.CheckPermissionRequest,
) (*v1.CheckPermissionResponse, error) {
	var result v1.CheckPermissionResponse

	res, err := s.service.Resty.R().
		SetContext(ctx).
		SetBody(req).
		SetResult(&result).
		Post(CheckPermissionURL)
	if err != nil {
		return nil, fmt.Errorf("failed to check permission: %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to check permission: %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	return &result, nil
}

// GetUserRoles retrieves the roles assigned to a user.
func (s *RBACService) GetUserRoles(
	ctx context.Context,
	userID string,
) (*v1.ListResponse[v1.Role], error) {
	var result v1.ListResponse[v1.Role]

	res, err := s.service.Resty.R().
		SetContext(ctx).
		SetPathParam("id", userID).
		SetResult(&result).
		Get(GetUserRolesURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to get user roles: %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	return &result, nil
}

// GetUserPermissions retrieves the permissions of a user.
func (s *RBACService) GetUserPermissions(
	ctx context.Context,
	userID string,
) (*v1.ListResponse[v1.Permission], error) {
	var result v1.ListResponse[v1.Permission]

	res, err := s.service.Resty.R().
		SetContext(ctx).
		SetPathParam("id", userID).
		SetResult(&result).
		Get(GetUserPermissionsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to get user permissions: %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	return &result, nil
}

// SyncPolicies synchronizes RBAC policies.
func (s *RBACService) SyncPolicies(ctx context.Context) error {
	res, err := s.service.Resty.R().
		SetContext(ctx).
		Post(SyncPoliciesURL)
	if err != nil {
		return fmt.Errorf("failed to sync policies: %w", err)
	}

	if res.IsError() {
		return fmt.Errorf("failed to sync policies: %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	return nil
}
