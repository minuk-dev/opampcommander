package port

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/serverevent"
)

// AgentPersistencePort is an interface that defines the methods for agent persistence.
type AgentPersistencePort interface {
	// GetAgent retrieves an agent by its instance UID.
	GetAgent(ctx context.Context, instanceUID uuid.UUID) (*model.Agent, error)
	// GetAgentByID retrieves an agent by its ID.
	PutAgent(ctx context.Context, agent *model.Agent) error
	// ListAgents retrieves a list of agents with pagination options.
	ListAgents(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.Agent], error)
	// ListAgentsBySelector retrieves a list of agents matching the given selector with pagination options.
	ListAgentsBySelector(
		ctx context.Context,
		selector model.AgentSelector,
		options *model.ListOptions,
	) (*model.ListResponse[*model.Agent], error)
	// SearchAgents searches agents by query with pagination options.
	SearchAgents(ctx context.Context, query string, options *model.ListOptions) (*model.ListResponse[*model.Agent], error)
}

// ServerEventSenderPort is an interface that defines the methods for sending events to servers.
type ServerEventSenderPort interface {
	// SendMessageToServer sends a message to the specified server.
	SendMessageToServer(ctx context.Context, serverID string, message serverevent.Message) error
}

// ReceiveServerEventHandler is a function type for handling received server events.
type ReceiveServerEventHandler func(ctx context.Context, message *serverevent.Message) error

// ServerEventReceiverPort is an interface that defines the methods for receiving events from servers.
type ServerEventReceiverPort interface {
	// StartReceiver starts receiving messages from servers using the provided handler.
	// It's a blocking call.
	StartReceiver(ctx context.Context, handler ReceiveServerEventHandler) error
}

// AgentGroupPersistencePort is an interface that defines the methods for agent group persistence.
type AgentGroupPersistencePort interface {
	// GetAgentGroup retrieves an agent group by its ID.
	GetAgentGroup(ctx context.Context, name string, options *model.GetOptions) (*model.AgentGroup, error)
	// PutAgentGroup saves the agent group.
	PutAgentGroup(ctx context.Context, name string, agentGroup *model.AgentGroup) (*model.AgentGroup, error)
	// ListAgentGroups retrieves a list of agent groups with pagination options.
	ListAgentGroups(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.AgentGroup], error)
}

// ServerPersistencePort is an interface that defines the methods for server persistence.
type ServerPersistencePort interface {
	// GetServer retrieves a server by its ID.
	GetServer(ctx context.Context, id string) (*model.Server, error)
	// PutServer saves or updates a server.
	PutServer(ctx context.Context, server *model.Server) error
	// ListServers retrieves a list of all servers.
	ListServers(ctx context.Context) ([]*model.Server, error)
}

// AgentPackagePersistencePort is an interface that defines the methods for agent package persistence.
type AgentPackagePersistencePort interface {
	// GetAgentPackage retrieves an agent package by its name.
	GetAgentPackage(ctx context.Context, name string) (*model.AgentPackage, error)
	// PutAgentPackage saves or updates an agent package.
	PutAgentPackage(ctx context.Context, agentPackage *model.AgentPackage) (*model.AgentPackage, error)
	// ListAgentPackages retrieves a list of agent packages with pagination options.
	ListAgentPackages(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.AgentPackage], error)
}

// AgentRemoteConfigPersistencePort is an interface that defines the methods for agent remote config persistence.
type AgentRemoteConfigPersistencePort interface {
	// GetAgentRemoteConfig retrieves an agent remote config by its name.
	GetAgentRemoteConfig(ctx context.Context, name string) (*model.AgentRemoteConfig, error)
	// PutAgentRemoteConfig saves or updates an agent remote config.
	PutAgentRemoteConfig(
		ctx context.Context,
		config *model.AgentRemoteConfig,
	) (*model.AgentRemoteConfig, error)
	// ListAgentRemoteConfigs retrieves a list of agent remote configs with pagination options.
	ListAgentRemoteConfigs(
		ctx context.Context,
		options *model.ListOptions,
	) (*model.ListResponse[*model.AgentRemoteConfig], error)
}

// CertificatePersistencePort is an interface that defines the methods for ceritificate config persistence.
type CertificatePersistencePort interface {
	GetCertificate(ctx context.Context, name string) (*model.Certificate, error)
	PutCertificate(ctx context.Context, certificate *model.Certificate) (*model.Certificate, error)
	ListCertificate(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.Certificate], error)
}

// UserPersistencePort is an interface that defines the methods for user persistence.
type UserPersistencePort interface {
	// GetUser retrieves a user by their UID.
	GetUser(ctx context.Context, uid uuid.UUID) (*model.User, error)
	// GetUserByEmail retrieves a user by their email.
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	// PutUser saves or updates a user.
	PutUser(ctx context.Context, user *model.User) (*model.User, error)
	// ListUsers retrieves a list of users with pagination options.
	ListUsers(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.User], error)
	// DeleteUser deletes a user by their UID.
	DeleteUser(ctx context.Context, uid uuid.UUID) error
}

// RolePersistencePort is an interface that defines the methods for role persistence.
type RolePersistencePort interface {
	// GetRole retrieves a role by its UID.
	GetRole(ctx context.Context, uid uuid.UUID) (*model.Role, error)
	// GetRoleByName retrieves a role by its display name.
	GetRoleByName(ctx context.Context, displayName string) (*model.Role, error)
	// PutRole saves or updates a role.
	PutRole(ctx context.Context, role *model.Role) (*model.Role, error)
	// ListRoles retrieves a list of roles with pagination options.
	ListRoles(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.Role], error)
	// DeleteRole deletes a role by its UID.
	DeleteRole(ctx context.Context, uid uuid.UUID) error
}

// PermissionPersistencePort is an interface that defines the methods for permission persistence.
type PermissionPersistencePort interface {
	// GetPermission retrieves a permission by its UID.
	GetPermission(ctx context.Context, uid uuid.UUID) (*model.Permission, error)
	// GetPermissionByName retrieves a permission by its name (e.g., "agent:read").
	GetPermissionByName(ctx context.Context, name string) (*model.Permission, error)
	// PutPermission saves or updates a permission.
	PutPermission(ctx context.Context, permission *model.Permission) (*model.Permission, error)
	// ListPermissions retrieves a list of permissions with pagination options.
	ListPermissions(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.Permission], error)
	// DeletePermission deletes a permission by its UID.
	DeletePermission(ctx context.Context, uid uuid.UUID) error
}

// UserRolePersistencePort is an interface that defines the methods for user role persistence.
type UserRolePersistencePort interface {
	// GetUserRole retrieves a user role assignment by its UID.
	GetUserRole(ctx context.Context, uid uuid.UUID) (*model.UserRole, error)
	// GetUserRoleByUserAndRole retrieves a user role assignment by user and role IDs.
	GetUserRoleByUserAndRole(ctx context.Context, userID, roleID uuid.UUID) (*model.UserRole, error)
	// PutUserRole saves or updates a user role assignment.
	PutUserRole(ctx context.Context, userRole *model.UserRole) (*model.UserRole, error)
	// ListUserRoles retrieves a list of user role assignments with pagination options.
	ListUserRoles(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.UserRole], error)
	// GetUserRoles retrieves all roles assigned to a user.
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*model.Role, error)
	// GetRoleUsers retrieves all users assigned to a role.
	GetRoleUsers(ctx context.Context, roleID uuid.UUID) ([]*model.User, error)
	// DeleteUserRole deletes a user role assignment by its UID.
	DeleteUserRole(ctx context.Context, uid uuid.UUID) error
	// DeleteUserRoles deletes all role assignments for a user.
	DeleteUserRoles(ctx context.Context, userID uuid.UUID) error
	// DeleteRoleUsers deletes all user assignments for a role.
	DeleteRoleUsers(ctx context.Context, roleID uuid.UUID) error
}

// RBACEnforcerPort is an interface that defines the methods for Casbin enforcer operations.
type RBACEnforcerPort interface {
	// CheckPermission checks if a user has a specific permission for a resource and action.
	CheckPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error)
	// LoadPolicy loads all policies from the persistence storage into the enforcer.
	LoadPolicy(ctx context.Context) error
	// SavePolicy saves the policies to the persistence storage.
	SavePolicy(ctx context.Context) error
	// AddGroupingPolicy adds a grouping (role) policy to the enforcer.
	// e.g., AddGroupingPolicy(userID, roleID)
	AddGroupingPolicy(ctx context.Context, params ...interface{}) (bool, error)
	// RemoveGroupingPolicy removes a grouping (role) policy from the enforcer.
	RemoveGroupingPolicy(ctx context.Context, params ...interface{}) (bool, error)
	// GetGroupingPolicy gets all grouping policies.
	GetGroupingPolicy() [][]string
	// AddNamedPolicy adds a named policy to the enforcer.
	AddNamedPolicy(ctx context.Context, ptype string, params ...interface{}) (bool, error)
	// RemoveNamedPolicy removes a named policy from the enforcer.
	RemoveNamedPolicy(ctx context.Context, ptype string, params ...interface{}) (bool, error)
	// GetNamedPolicy gets all named policies.
	GetNamedPolicy(ptype string) [][]string
}

// OrgRoleMappingPersistencePort is an interface for org-role mapping persistence.
type OrgRoleMappingPersistencePort interface {
	// GetOrgRoleMapping retrieves an org-role mapping by its UID.
	GetOrgRoleMapping(ctx context.Context, uid uuid.UUID) (*model.OrgRoleMapping, error)
	// PutOrgRoleMapping saves or updates an org-role mapping.
	PutOrgRoleMapping(ctx context.Context, mapping *model.OrgRoleMapping) (*model.OrgRoleMapping, error)
	// ListOrgRoleMappings retrieves a list of org-role mappings.
	ListOrgRoleMappings(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.OrgRoleMapping], error)
	// ListOrgRoleMappingsByProvider retrieves mappings for a specific provider.
	ListOrgRoleMappingsByProvider(ctx context.Context, provider string) ([]*model.OrgRoleMapping, error)
	// DeleteOrgRoleMapping deletes an org-role mapping.
	DeleteOrgRoleMapping(ctx context.Context, uid uuid.UUID) error
}

var (
	// ErrResourceNotExist is an error that indicates that the resource does not exist.
	ErrResourceNotExist = errors.New("resource does not exist")
	// ErrMultipleResourceExist is an error that indicates that multiple resources exist.
	ErrMultipleResourceExist = errors.New("multiple resources exist")
)
