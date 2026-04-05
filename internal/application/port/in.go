// Package port is a package that defines the ports for the application layer.
package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	opamptypes "github.com/open-telemetry/opamp-go/server/types"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/agent/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

// OpAMPUsecase is a use case that handles OpAMP protocol operations.
// Please see [github.com/open-telemetry/opamp-go/server/types/ConnectionCallbacks].
type OpAMPUsecase interface {
	OnConnected(ctx context.Context, conn opamptypes.Connection)
	OnConnectedWithType(ctx context.Context, conn opamptypes.Connection, isWebSocket bool)
	OnMessage(ctx context.Context, conn opamptypes.Connection, message *protobufs.AgentToServer) *protobufs.ServerToAgent
	OnConnectionClose(conn opamptypes.Connection)
	OnReadMessageError(conn opamptypes.Connection, mt int, msgByte []byte, err error)
	OnMessageResponseError(conn opamptypes.Connection, message *protobufs.ServerToAgent, err error)
}

// AdminUsecase is a use case that handles admin operations.
type AdminUsecase interface {
	ListConnections(ctx context.Context, namespace string,
		options *model.ListOptions) (*model.ListResponse[*agentmodel.Connection], error)
}

// NamespaceManageUsecase is a use case that handles namespace management operations.
type NamespaceManageUsecase interface {
	GetNamespace(ctx context.Context, name string) (*v1.Namespace, error)
	ListNamespaces(ctx context.Context,
		options *model.ListOptions) (*v1.ListResponse[v1.Namespace], error)
	CreateNamespace(ctx context.Context,
		namespace *v1.Namespace) (*v1.Namespace, error)
	UpdateNamespace(ctx context.Context, name string,
		namespace *v1.Namespace) (*v1.Namespace, error)
	DeleteNamespace(ctx context.Context, name string) error
}

// AgentPackageManageUsecase is a use case that handles agent package operations.
type AgentPackageManageUsecase interface {
	GetAgentPackage(ctx context.Context, namespace string, name string) (*v1.AgentPackage, error)
	ListAgentPackages(ctx context.Context, options *model.ListOptions) (*v1.ListResponse[v1.AgentPackage], error)
	CreateAgentPackage(ctx context.Context, agentPackage *v1.AgentPackage) (*v1.AgentPackage, error)
	UpdateAgentPackage(ctx context.Context, namespace string, name string,
		agentPackage *v1.AgentPackage) (*v1.AgentPackage, error)
	DeleteAgentPackage(ctx context.Context, namespace string, name string) error
}

// AgentManageUsecase is a use case that handles agent management operations.
type AgentManageUsecase interface {
	GetAgent(ctx context.Context, namespace string, instanceUID uuid.UUID) (*v1.Agent, error)
	ListAgents(ctx context.Context, namespace string,
		options *model.ListOptions) (*v1.ListResponse[v1.Agent], error)
	SearchAgents(ctx context.Context, namespace string, query string,
		options *model.ListOptions) (*v1.ListResponse[v1.Agent], error)
	UpdateAgent(ctx context.Context, namespace string, instanceUID uuid.UUID,
		agent *v1.Agent) (*v1.Agent, error)
}

// AgentGroupManageUsecase is a use case that handles agent group management operations.
type AgentGroupManageUsecase interface {
	GetAgentGroup(ctx context.Context, namespace string, name string,
		options *model.GetOptions) (*v1.AgentGroup, error)
	ListAgentGroups(ctx context.Context, options *model.ListOptions) (*v1.ListResponse[v1.AgentGroup], error)
	ListAgentsByAgentGroup(
		ctx context.Context,
		namespace string,
		agentGroupName string,
		options *model.ListOptions,
	) (*v1.ListResponse[v1.Agent], error)
	CreateAgentGroup(ctx context.Context, agentGroup *v1.AgentGroup) (*v1.AgentGroup, error)
	UpdateAgentGroup(ctx context.Context, namespace string, name string,
		agentGroup *v1.AgentGroup) (*v1.AgentGroup, error)
	DeleteAgentGroup(ctx context.Context, namespace string, name string) error
}

// CertificateManageUsecase is a use case that handles certificate management operations.
type CertificateManageUsecase interface {
	GetCertificate(ctx context.Context, namespace string, name string) (*v1.Certificate, error)
	ListCertificates(ctx context.Context, options *model.ListOptions) (*v1.ListResponse[v1.Certificate], error)
	CreateCertificate(ctx context.Context, certificate *v1.Certificate) (*v1.Certificate, error)
	UpdateCertificate(ctx context.Context, namespace string, name string,
		certificate *v1.Certificate) (*v1.Certificate, error)
	DeleteCertificate(ctx context.Context, namespace string, name string) error
}

// AgentRemoteConfigManageUsecase is a use case that handles agent remote config management operations.
type AgentRemoteConfigManageUsecase interface {
	GetAgentRemoteConfig(ctx context.Context, namespace string,
		name string) (*v1.AgentRemoteConfig, error)
	ListAgentRemoteConfigs(ctx context.Context,
		options *model.ListOptions) (*v1.ListResponse[v1.AgentRemoteConfig], error)
	CreateAgentRemoteConfig(ctx context.Context,
		agentRemoteConfig *v1.AgentRemoteConfig) (*v1.AgentRemoteConfig, error)
	UpdateAgentRemoteConfig(ctx context.Context, namespace string, name string,
		agentRemoteConfig *v1.AgentRemoteConfig) (*v1.AgentRemoteConfig, error)
	DeleteAgentRemoteConfig(ctx context.Context, namespace string, name string) error
}

// UserManageUsecase is a use case that handles user management operations.
type UserManageUsecase interface {
	GetUser(ctx context.Context, uid uuid.UUID) (*v1.User, error)
	GetUserByEmail(ctx context.Context, email string) (*v1.User, error)
	ListUsers(ctx context.Context, options *model.ListOptions) (*v1.ListResponse[v1.User], error)
	CreateUser(ctx context.Context, user *v1.User) (*v1.User, error)
	DeleteUser(ctx context.Context, uid uuid.UUID) error
	GetUserProfile(ctx context.Context, email string) (*v1.UserProfileResponse, error)
}

// RoleManageUsecase is a use case that handles role management operations.
type RoleManageUsecase interface {
	GetRole(ctx context.Context, uid uuid.UUID) (*v1.Role, error)
	ListRoles(ctx context.Context, options *model.ListOptions) (*v1.ListResponse[v1.Role], error)
	CreateRole(ctx context.Context, role *v1.Role) (*v1.Role, error)
	UpdateRole(ctx context.Context, uid uuid.UUID, role *v1.Role) (*v1.Role, error)
	DeleteRole(ctx context.Context, uid uuid.UUID) error
}

// PermissionManageUsecase is a use case that handles permission management operations.
type PermissionManageUsecase interface {
	GetPermission(ctx context.Context, uid uuid.UUID) (*v1.Permission, error)
	ListPermissions(ctx context.Context, options *model.ListOptions) (*v1.ListResponse[v1.Permission], error)
}

// RBACManageUsecase is a use case that handles RBAC management operations.
type RBACManageUsecase interface {
	AssignRole(ctx context.Context, req *v1.AssignRoleRequest) error
	UnassignRole(ctx context.Context, userID, roleID uuid.UUID) error
	CheckPermission(ctx context.Context, req *v1.CheckPermissionRequest) (*v1.CheckPermissionResponse, error)
	GetUserRoles(ctx context.Context, userID uuid.UUID) (*v1.ListResponse[v1.Role], error)
	GetUserPermissions(ctx context.Context, userID uuid.UUID) (*v1.ListResponse[v1.Permission], error)
	SyncPolicies(ctx context.Context) error
}
