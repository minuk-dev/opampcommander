package userport

import (
	"context"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	usermodel "github.com/minuk-dev/opampcommander/internal/domain/user/model"
)

// UserPersistencePort is an interface that defines the methods for user persistence.
type UserPersistencePort interface {
	// GetUser retrieves a user by their UID.
	GetUser(ctx context.Context, uid uuid.UUID) (*usermodel.User, error)
	// GetUserByEmail retrieves a user by their email.
	GetUserByEmail(ctx context.Context, email string) (*usermodel.User, error)
	// PutUser saves or updates a user.
	PutUser(ctx context.Context, user *usermodel.User) (*usermodel.User, error)
	// ListUsers retrieves a list of users with pagination options.
	ListUsers(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*usermodel.User], error)
	// DeleteUser deletes a user by their UID.
	DeleteUser(ctx context.Context, uid uuid.UUID) error
}

// RolePersistencePort is an interface that defines the methods for role persistence.
type RolePersistencePort interface {
	// GetRole retrieves a role by its UID.
	GetRole(ctx context.Context, uid uuid.UUID) (*usermodel.Role, error)
	// GetRoleByName retrieves a role by its display name.
	GetRoleByName(ctx context.Context, displayName string) (*usermodel.Role, error)
	// PutRole saves or updates a role.
	PutRole(ctx context.Context, role *usermodel.Role) (*usermodel.Role, error)
	// ListRoles retrieves a list of roles with pagination options.
	ListRoles(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*usermodel.Role], error)
	// DeleteRole deletes a role by its UID.
	DeleteRole(ctx context.Context, uid uuid.UUID) error
}

// PermissionPersistencePort is an interface that defines the methods for permission persistence.
type PermissionPersistencePort interface {
	// GetPermission retrieves a permission by its UID.
	GetPermission(ctx context.Context, uid uuid.UUID) (*usermodel.Permission, error)
	// GetPermissionByName retrieves a permission by its name (e.g., "agent:read").
	GetPermissionByName(ctx context.Context, name string) (*usermodel.Permission, error)
	// PutPermission saves or updates a permission.
	PutPermission(ctx context.Context, permission *usermodel.Permission) (*usermodel.Permission, error)
	// ListPermissions retrieves a list of permissions with pagination options.
	ListPermissions(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*usermodel.Permission], error)
	// DeletePermission deletes a permission by its UID.
	DeletePermission(ctx context.Context, uid uuid.UUID) error
}

// UserRolePersistencePort is an interface that defines the methods for user role persistence.
type UserRolePersistencePort interface {
	// GetUserRole retrieves a user role assignment by its UID.
	GetUserRole(ctx context.Context, uid uuid.UUID) (*usermodel.UserRole, error)
	// GetUserRoleByUserAndRole retrieves a user role assignment by user, role, and namespace.
	GetUserRoleByUserAndRole(ctx context.Context, userID, roleID uuid.UUID, namespace string) (*usermodel.UserRole, error)
	// PutUserRole saves or updates a user role assignment.
	PutUserRole(ctx context.Context, userRole *usermodel.UserRole) (*usermodel.UserRole, error)
	// ListUserRoles retrieves a list of user role assignments with pagination options.
	ListUserRoles(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*usermodel.UserRole], error)
	// GetUserRoles retrieves all roles assigned to a user.
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*usermodel.Role, error)
	// GetUserRolesInNamespace retrieves roles assigned to a user in a specific namespace.
	// Returns roles where namespace matches exactly or is "*" (all namespaces).
	GetUserRolesInNamespace(ctx context.Context, userID uuid.UUID, namespace string) ([]*usermodel.Role, error)
	// GetRoleUsers retrieves all users assigned to a role.
	GetRoleUsers(ctx context.Context, roleID uuid.UUID) ([]*usermodel.User, error)
	// DeleteUserRole deletes a user role assignment by its UID.
	DeleteUserRole(ctx context.Context, uid uuid.UUID) error
	// DeleteUserRoles deletes all role assignments for a user.
	DeleteUserRoles(ctx context.Context, userID uuid.UUID) error
	// DeleteRoleUsers deletes all user assignments for a role.
	DeleteRoleUsers(ctx context.Context, roleID uuid.UUID) error
}

// RoleBindingPersistencePort is an interface that defines the methods for role binding persistence.
type RoleBindingPersistencePort interface {
	// GetRoleBinding retrieves a role binding by namespace and name.
	GetRoleBinding(ctx context.Context, namespace, name string) (*usermodel.RoleBinding, error)
	// PutRoleBinding saves or updates a role binding.
	PutRoleBinding(ctx context.Context, rb *usermodel.RoleBinding) (*usermodel.RoleBinding, error)
	// ListRoleBindings retrieves a list of role bindings with pagination options.
	ListRoleBindings(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*usermodel.RoleBinding], error)
	// DeleteRoleBinding deletes a role binding by namespace and name.
	DeleteRoleBinding(ctx context.Context, namespace, name string) error
}

// RBACEnforcerPort is an interface that defines the methods for Casbin enforcer operations.
//
//nolint:interfacebloat // Casbin enforcer adapter naturally requires many operations.
type RBACEnforcerPort interface {
	// CheckPermission checks if a user has a specific permission for a resource and action in a namespace.
	CheckPermission(ctx context.Context, sub, dom, obj, act string) (bool, error)
	// LoadPolicy loads all policies from the persistence storage into the enforcer.
	LoadPolicy(ctx context.Context) error
	// SavePolicy saves the policies to the persistence storage.
	SavePolicy(ctx context.Context) error
	// AddGroupingPolicy adds a grouping (role) policy to the enforcer.
	AddGroupingPolicy(ctx context.Context, params ...any) (bool, error)
	// RemoveGroupingPolicy removes a grouping (role) policy from the enforcer.
	RemoveGroupingPolicy(ctx context.Context, params ...any) (bool, error)
	// GetGroupingPolicy gets all grouping policies.
	GetGroupingPolicy() ([][]string, error)
	// AddNamedPolicy adds a named policy to the enforcer.
	AddNamedPolicy(ctx context.Context, ptype string, params ...any) (bool, error)
	// RemoveNamedPolicy removes a named policy from the enforcer.
	RemoveNamedPolicy(ctx context.Context, ptype string, params ...any) (bool, error)
	// GetNamedPolicy gets all named policies.
	GetNamedPolicy(ptype string) ([][]string, error)
	// ClearPolicy removes all policies from the enforcer.
	ClearPolicy(ctx context.Context)
	// BuildRoleLinks rebuilds the role inheritance links after policy changes.
	BuildRoleLinks(ctx context.Context) error
}

