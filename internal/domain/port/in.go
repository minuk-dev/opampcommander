package port

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/serverevent"
)

var (
	// ErrConnectionAlreadyExists is an error that indicates that the connection already exists.
	ErrConnectionAlreadyExists = errors.New("connection already exists")
	// ErrConnectionNotFound is an error that indicates that the connection was not found.
	ErrConnectionNotFound = errors.New("connection not found")
)

// AgentUsecase is an interface that defines the methods for agent use cases.
type AgentUsecase interface {
	// GetAgent retrieves an agent by its instance UID.
	GetAgent(ctx context.Context, instanceUID uuid.UUID) (*model.Agent, error)
	// GetOrCreateAgent retrieves an agent by its instance UID or creates a new one if it does not exist.
	GetOrCreateAgent(ctx context.Context, instanceUID uuid.UUID) (*model.Agent, error)
	// ListAgentsBySelector lists agents by the given selector.
	ListAgentsBySelector(
		ctx context.Context,
		selector model.AgentSelector,
		options *model.ListOptions,
	) (*model.ListResponse[*model.Agent], error)
	// SaveAgent saves the agent.
	SaveAgent(ctx context.Context, agent *model.Agent) error
	// ListAgents lists all agents.
	ListAgents(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.Agent], error)
	// SearchAgents searches agents by instance UID prefix.
	SearchAgents(ctx context.Context, query string, options *model.ListOptions) (*model.ListResponse[*model.Agent], error)
}

// AgentNotificationUsecase is an interface for notifying servers about agent changes.
type AgentNotificationUsecase interface {
	// NotifyAgentUpdated notifies the connected server that the agent has pending messages.
	NotifyAgentUpdated(ctx context.Context, agent *model.Agent) error
	// RestartAgent requests the agent to restart.
	RestartAgent(ctx context.Context, instanceUID uuid.UUID) error
}

// AgentPackageUsecase is an interface that defines the methods for agent package use cases.
type AgentPackageUsecase interface {
	// GetAgentPackage retrieves an agent package by its name.
	GetAgentPackage(ctx context.Context, name string) (*model.AgentPackage, error)
	// ListAgentPackages lists all agent packages.
	ListAgentPackages(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.AgentPackage], error)
	// SaveAgentPackage saves the agent package.
	SaveAgentPackage(ctx context.Context, agentPackage *model.AgentPackage) (*model.AgentPackage, error)
	// DeleteAgentPackage deletes the agent package by its name.
	DeleteAgentPackage(ctx context.Context, name string, deletedAt time.Time, deletedBy string) error
}

// AgentRemoteConfigUsecase is an interface that defines the methods for agent remote config use cases.
type AgentRemoteConfigUsecase interface {
	// GetAgentRemoteConfig retrieves an agent remote config by its name.
	GetAgentRemoteConfig(ctx context.Context, name string) (*model.AgentRemoteConfig, error)
	// ListAgentRemoteConfigs lists all agent remote configs.
	ListAgentRemoteConfigs(
		ctx context.Context, options *model.ListOptions,
	) (*model.ListResponse[*model.AgentRemoteConfig], error)
	// SaveAgentRemoteConfig saves the agent remote config.
	SaveAgentRemoteConfig(
		ctx context.Context, agentRemoteConfig *model.AgentRemoteConfig,
	) (*model.AgentRemoteConfig, error)
	// DeleteAgentRemoteConfig deletes the agent remote config by its name.
	DeleteAgentRemoteConfig(ctx context.Context, name string, deletedAt time.Time, deletedBy string) error
}

// AgentGroupUsecase is an interface that defines the methods for agent group use cases.
type AgentGroupUsecase interface {
	// GetAgentGroup retrieves an agent group by its name
	GetAgentGroup(ctx context.Context, name string, options *model.GetOptions) (*model.AgentGroup, error)
	// SaveAgentGroup saves the agent group.
	ListAgentGroups(
		ctx context.Context, options *model.ListOptions,
	) (*model.ListResponse[*model.AgentGroup], error)
	// SaveAgentGroup saves the agent group.
	SaveAgentGroup(ctx context.Context, name string, agentGroup *model.AgentGroup) (*model.AgentGroup, error)
	// DeleteAgentGroup deletes the agent group by its ID.
	DeleteAgentGroup(ctx context.Context, name string, deletedAt time.Time, deletedBy string) error
	// GetAgentGroupsForAgent retrieves all agent groups that match the agent's attributes.
	GetAgentGroupsForAgent(ctx context.Context, agent *model.Agent) ([]*model.AgentGroup, error)
}

// AgentGroupRelatedUsecase is an interface that defines methods related to agent groups.
type AgentGroupRelatedUsecase interface {
	// ListAgentsByAgentGroup lists agents belonging to a specific agent group.
	ListAgentsByAgentGroup(
		ctx context.Context,
		agentGroup *model.AgentGroup,
		options *model.ListOptions,
	) (*model.ListResponse[*model.Agent], error)
}

// CertificateUsecase defines the interface for certificate use cases.
type CertificateUsecase interface {
	GetCertificate(ctx context.Context, name string) (*model.Certificate, error)
	SaveCertificate(ctx context.Context, certificate *model.Certificate) (*model.Certificate, error)
	ListCertificate(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.Certificate], error)
	DeleteCertificate(ctx context.Context, name string, deletedAt time.Time, deletedBy string) (*model.Certificate, error)
}

// ServerUsecase is an interface that defines the methods for server use cases.
type ServerUsecase interface {
	// ServerUsecase should also implement ServerMessageUsecase
	ServerMessageUsecase
	// GetServer retrieves a server by its ID.
	GetServer(ctx context.Context, id string) (*model.Server, error)
	// ListServers lists all servers.
	// The number of servers is expected to be small, so no pagination is needed.
	ListServers(ctx context.Context) ([]*model.Server, error)
}

// ServerIdentityProvider is an interface that defines the methods for providing server identity.
type ServerIdentityProvider interface {
	// CurrentServer returns the current server.
	CurrentServer(ctx context.Context) (*model.Server, error)
}

// ServerMessageUsecase is an interface that defines the methods for server message use cases.
// Some usecases may require sending messages to other servers.
// So, this interface defines as a separate interface.
type ServerMessageUsecase interface {
	// SendMessageToServerByServerID sends a message to the specified server.
	SendMessageToServerByServerID(ctx context.Context, serverID string, message serverevent.Message) error
	// SendMessageToServer sends a message to the specified server.
	SendMessageToServer(ctx context.Context, server *model.Server, message serverevent.Message) error
}

// ServerReceiverUsecase is an interface that defines the methods for server receiver use cases.
type ServerReceiverUsecase interface {
	// ReceiveMessageFromServer processes a message received from a server.
	ReceiveMessageFromServer() error
}

// ConnectionUsecase is an interface that defines the methods for connection use cases.
type ConnectionUsecase interface {
	// GetConnectionByInstanceUID returns the connection for the given instance UID.
	GetConnectionByInstanceUID(ctx context.Context, instanceUID uuid.UUID) (*model.Connection, error)
	// GetOrCreateConnectionByID returns the connection for the given ID or creates a new one if it does not exist.
	GetOrCreateConnectionByID(ctx context.Context, id any) (*model.Connection, error)
	// GetConnectionByID returns the connection for the given ID.
	GetConnectionByID(ctx context.Context, id any) (*model.Connection, error)
	// ListConnections returns the list of connections.
	ListConnections(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.Connection], error)
	// SaveConnection saves the connection.
	SaveConnection(ctx context.Context, connection *model.Connection) error
	// DeleteConnection deletes the connection.
	DeleteConnection(ctx context.Context, connection *model.Connection) error
	// SendServerToAgent sends a ServerToAgent message to the agent via WebSocket connection.
	SendServerToAgent(ctx context.Context, instanceUID uuid.UUID, message *protobufs.ServerToAgent) error
}

// UserUsecase is an interface that defines the methods for user use cases.
type UserUsecase interface {
	// GetUser retrieves a user by their UID.
	GetUser(ctx context.Context, uid uuid.UUID) (*model.User, error)
	// GetUserByEmail retrieves a user by their email.
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	// ListUsers lists all users.
	ListUsers(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.User], error)
	// SaveUser saves the user.
	SaveUser(ctx context.Context, user *model.User) error
	// DeleteUser deletes the user.
	DeleteUser(ctx context.Context, uid uuid.UUID) error
}

// RoleUsecase is an interface that defines the methods for role use cases.
type RoleUsecase interface {
	// GetRole retrieves a role by its UID.
	GetRole(ctx context.Context, uid uuid.UUID) (*model.Role, error)
	// GetRoleByName retrieves a role by its display name.
	GetRoleByName(ctx context.Context, displayName string) (*model.Role, error)
	// ListRoles lists all roles.
	ListRoles(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.Role], error)
	// SaveRole saves the role.
	SaveRole(ctx context.Context, role *model.Role) error
	// DeleteRole deletes the role (only if it's not built-in).
	DeleteRole(ctx context.Context, uid uuid.UUID) error
}

// PermissionUsecase is an interface that defines the methods for permission use cases.
type PermissionUsecase interface {
	// GetPermission retrieves a permission by its UID.
	GetPermission(ctx context.Context, uid uuid.UUID) (*model.Permission, error)
	// GetPermissionByName retrieves a permission by its name (e.g., "agent:read").
	GetPermissionByName(ctx context.Context, name string) (*model.Permission, error)
	// ListPermissions lists all permissions.
	ListPermissions(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.Permission], error)
	// SavePermission saves the permission.
	SavePermission(ctx context.Context, permission *model.Permission) error
	// DeletePermission deletes the permission (only if it's not built-in).
	DeletePermission(ctx context.Context, uid uuid.UUID) error
}

// UserRoleUsecase is an interface that defines the methods for user role use cases.
type UserRoleUsecase interface {
	// AssignRole assigns a role to a user.
	AssignRole(ctx context.Context, userID, roleID, assignedBy uuid.UUID) error
	// UnassignRole removes a role from a user.
	UnassignRole(ctx context.Context, userID, roleID uuid.UUID) error
	// GetUserRoles returns all roles assigned to a user.
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*model.Role, error)
	// GetRoleUsers returns all users assigned to a role.
	GetRoleUsers(ctx context.Context, roleID uuid.UUID) ([]*model.User, error)
	// ListUserRoles lists all user role assignments.
	ListUserRoles(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.UserRole], error)
}

// RBACUsecase is an interface that defines RBAC authorization methods.
type RBACUsecase interface {
	// CheckPermission checks if a user has a specific permission.
	CheckPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error)
	// GetUserPermissions returns all permissions available to a user through their roles.
	GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]*model.Permission, error)
	// GetEffectivePermissions returns all effective permissions for a user (including inherited).
	GetEffectivePermissions(ctx context.Context, userID uuid.UUID) ([]*model.Permission, error)
	// SyncPolicies synchronizes RBAC policies with the Casbin enforcer.
	SyncPolicies(ctx context.Context) error
}

// IdentityProviderUsecase is a provider-agnostic interface for resolving external identities.
// Each authentication provider (GitHub, Google, LDAP, etc.) implements this interface
// to translate provider-specific identity into the common ExternalIdentity model.
type IdentityProviderUsecase interface {
	// ProviderName returns the unique name of this identity provider (e.g., "github", "google").
	ProviderName() string
	// ResolveIdentity resolves an authenticated token/credential into an ExternalIdentity.
	ResolveIdentity(ctx context.Context, accessToken string) (*model.ExternalIdentity, error)
	// ListOrganizations returns the organizations/groups the user belongs to.
	ListOrganizations(ctx context.Context, accessToken string) ([]string, error)
}

// OrgRoleMappingUsecase manages mappings from external org/group memberships to internal roles.
type OrgRoleMappingUsecase interface {
	// GetOrgRoleMapping retrieves a mapping by its UID.
	GetOrgRoleMapping(ctx context.Context, uid uuid.UUID) (*model.OrgRoleMapping, error)
	// ListOrgRoleMappings lists all org-role mappings.
	ListOrgRoleMappings(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.OrgRoleMapping], error)
	// ListOrgRoleMappingsByProvider lists mappings for a specific provider.
	ListOrgRoleMappingsByProvider(ctx context.Context, provider string) ([]*model.OrgRoleMapping, error)
	// SaveOrgRoleMapping saves an org-role mapping.
	SaveOrgRoleMapping(ctx context.Context, mapping *model.OrgRoleMapping) error
	// DeleteOrgRoleMapping deletes an org-role mapping.
	DeleteOrgRoleMapping(ctx context.Context, uid uuid.UUID) error
	// ResolveRolesForIdentity resolves which roles should be assigned based on
	// an external identity's org/group memberships and the configured mappings.
	ResolveRolesForIdentity(ctx context.Context, identity *model.ExternalIdentity) ([]*model.Role, error)
}
