package model

// Built-in permissions constants
const (
	// Agent permissions
	PermissionAgentRead   = "agent:read"
	PermissionAgentWrite  = "agent:write"
	PermissionAgentDelete = "agent:delete"
	PermissionAgentExecute = "agent:execute"

	// Agent Group permissions
	PermissionAgentGroupRead   = "agentgroup:read"
	PermissionAgentGroupWrite  = "agentgroup:write"
	PermissionAgentGroupDelete = "agentgroup:delete"

	// Agent Package permissions
	PermissionAgentPackageRead   = "agentpackage:read"
	PermissionAgentPackageWrite  = "agentpackage:write"
	PermissionAgentPackageDelete = "agentpackage:delete"

	// Agent Remote Config permissions
	PermissionAgentRemoteConfigRead   = "agentremoteconfig:read"
	PermissionAgentRemoteConfigWrite  = "agentremoteconfig:write"
	PermissionAgentRemoteConfigDelete = "agentremoteconfig:delete"

	// Certificate permissions
	PermissionCertificateRead   = "certificate:read"
	PermissionCertificateWrite  = "certificate:write"
	PermissionCertificateDelete = "certificate:delete"

	// Server permissions
	PermissionServerRead   = "server:read"
	PermissionServerWrite  = "server:write"
	PermissionServerDelete = "server:delete"

	// User management permissions
	PermissionUserRead   = "user:read"
	PermissionUserWrite  = "user:write"
	PermissionUserDelete = "user:delete"

	// Role management permissions
	PermissionRoleRead   = "role:read"
	PermissionRoleWrite  = "role:write"
	PermissionRoleDelete = "role:delete"

	// Permission management permissions
	PermissionPermissionRead   = "permission:read"
	PermissionPermissionWrite  = "permission:write"
	PermissionPermissionDelete = "permission:delete"
)

// Built-in role names
const (
	RoleAdmin       = "Admin"
	RoleAgentMgr    = "AgentManager"
	RoleConfigMgr   = "ConfigurationManager"
	RoleViewer      = "Viewer"
)

// Resources
const (
	ResourceAgent             = "agent"
	ResourceAgentGroup        = "agentgroup"
	ResourceAgentPackage      = "agentpackage"
	ResourceAgentRemoteConfig = "agentremoteconfig"
	ResourceCertificate       = "certificate"
	ResourceServer            = "server"
	ResourceUser              = "user"
	ResourceRole              = "role"
	ResourcePermission        = "permission"
)

// Actions
const (
	ActionRead   = "read"
	ActionWrite  = "write"
	ActionDelete = "delete"
	ActionExecute = "execute"
)
