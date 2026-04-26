package userport

import (
	"context"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	usermodel "github.com/minuk-dev/opampcommander/internal/domain/user/model"
)

// UserUsecase is an interface that defines the methods for user use cases.
type UserUsecase interface {
	// GetUser retrieves a user by their UID.
	GetUser(ctx context.Context, uid uuid.UUID) (*usermodel.User, error)
	// GetUserByEmail retrieves a user by their email.
	GetUserByEmail(ctx context.Context, email string) (*usermodel.User, error)
	// ListUsers lists all users.
	ListUsers(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*usermodel.User], error)
	// SaveUser saves the user.
	SaveUser(ctx context.Context, user *usermodel.User) error
	// DeleteUser deletes the user.
	DeleteUser(ctx context.Context, uid uuid.UUID) error
}

// RoleUsecase is an interface that defines the methods for role use cases.
type RoleUsecase interface {
	// GetRole retrieves a role by its UID.
	GetRole(ctx context.Context, uid uuid.UUID) (*usermodel.Role, error)
	// GetRoleByName retrieves a role by its display name.
	GetRoleByName(ctx context.Context, displayName string) (*usermodel.Role, error)
	// ListRoles lists all roles.
	ListRoles(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*usermodel.Role], error)
	// SaveRole saves the role.
	SaveRole(ctx context.Context, role *usermodel.Role) error
	// DeleteRole deletes the role (only if it's not built-in).
	DeleteRole(ctx context.Context, uid uuid.UUID) error
}

// PermissionUsecase is an interface that defines the methods for permission use cases.
type PermissionUsecase interface {
	// GetPermission retrieves a permission by its UID.
	GetPermission(ctx context.Context, uid uuid.UUID) (*usermodel.Permission, error)
	// GetPermissionByName retrieves a permission by its name (e.g., "agent:read").
	GetPermissionByName(ctx context.Context, name string) (*usermodel.Permission, error)
	// ListPermissions lists all permissions.
	ListPermissions(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*usermodel.Permission], error)
	// SavePermission saves the permission.
	SavePermission(ctx context.Context, permission *usermodel.Permission) error
	// DeletePermission deletes the permission (only if it's not built-in).
	DeletePermission(ctx context.Context, uid uuid.UUID) error
}

// UserRoleUsecase is an interface that defines the methods for user role use cases.
type UserRoleUsecase interface {
	// AssignRole assigns a role to a user in a specific namespace.
	// Use "*" as namespace for all namespaces.
	AssignRole(ctx context.Context, userID, roleID, assignedBy uuid.UUID, namespace string) error
	// UnassignRole removes a role from a user in a specific namespace.
	UnassignRole(ctx context.Context, userID, roleID uuid.UUID, namespace string) error
	// GetUserRoles returns all roles assigned to a user.
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*usermodel.Role, error)
	// GetUserRolesInNamespace returns roles assigned to a user in a specific namespace.
	GetUserRolesInNamespace(ctx context.Context, userID uuid.UUID, namespace string) ([]*usermodel.Role, error)
	// GetRoleUsers returns all users assigned to a role.
	GetRoleUsers(ctx context.Context, roleID uuid.UUID) ([]*usermodel.User, error)
	// ListUserRoles lists all user role assignments.
	ListUserRoles(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*usermodel.UserRole], error)
}

// RBACUsecase is an interface that defines RBAC authorization methods.
type RBACUsecase interface {
	// CheckPermission checks if a user has a specific permission in a namespace.
	CheckPermission(ctx context.Context, userID uuid.UUID, namespace, resource, action string) (bool, error)
	// GetUserPermissions returns all permissions available to a user through their roles.
	GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]*usermodel.Permission, error)
	// GetEffectivePermissions returns all effective permissions for a user (including inherited).
	GetEffectivePermissions(ctx context.Context, userID uuid.UUID) ([]*usermodel.Permission, error)
	// SyncPolicies synchronizes RBAC policies with the Casbin enforcer.
	SyncPolicies(ctx context.Context) error
}

// RoleBindingUsecase is an interface that defines the methods for role binding use cases.
type RoleBindingUsecase interface {
	// GetRoleBinding retrieves a role binding by namespace and name.
	GetRoleBinding(ctx context.Context, namespace, name string) (*usermodel.RoleBinding, error)
	// ListRoleBindings lists all role bindings.
	ListRoleBindings(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*usermodel.RoleBinding], error)
	// CreateRoleBinding creates a new role binding.
	CreateRoleBinding(ctx context.Context, rb *usermodel.RoleBinding) (*usermodel.RoleBinding, error)
	// UpdateRoleBinding updates an existing role binding.
	UpdateRoleBinding(ctx context.Context, namespace, name string,
		rb *usermodel.RoleBinding) (*usermodel.RoleBinding, error)
	// DeleteRoleBinding deletes a role binding by namespace and name.
	DeleteRoleBinding(ctx context.Context, namespace, name string) error
}

// IdentityProviderUsecase is a provider-agnostic interface for resolving external identities.
type IdentityProviderUsecase interface {
	// ProviderName returns the unique name of this identity provider.
	ProviderName() string
	// ResolveIdentity resolves an authenticated token/credential into an ExternalIdentity.
	ResolveIdentity(ctx context.Context, accessToken string) (*usermodel.ExternalIdentity, error)
	// ListOrganizations returns the organizations/groups the user belongs to.
	ListOrganizations(ctx context.Context, accessToken string) ([]string, error)
}

// OrgRoleMappingUsecase manages mappings from external org/group memberships to internal roles.
type OrgRoleMappingUsecase interface {
	// GetOrgRoleMapping retrieves a mapping by its UID.
	GetOrgRoleMapping(ctx context.Context, uid uuid.UUID) (*usermodel.OrgRoleMapping, error)
	// ListOrgRoleMappings lists all org-role mappings.
	ListOrgRoleMappings(ctx context.Context,
		options *model.ListOptions) (*model.ListResponse[*usermodel.OrgRoleMapping], error)
	// ListOrgRoleMappingsByProvider lists mappings for a specific provider.
	ListOrgRoleMappingsByProvider(ctx context.Context, provider string) ([]*usermodel.OrgRoleMapping, error)
	// SaveOrgRoleMapping saves an org-role mapping.
	SaveOrgRoleMapping(ctx context.Context, mapping *usermodel.OrgRoleMapping) error
	// DeleteOrgRoleMapping deletes an org-role mapping.
	DeleteOrgRoleMapping(ctx context.Context, uid uuid.UUID) error
	// ResolveRolesForIdentity resolves which roles should be assigned based on
	// an external identity's org/group memberships and the configured mappings.
	ResolveRolesForIdentity(ctx context.Context, identity *usermodel.ExternalIdentity) ([]*usermodel.Role, error)
}
