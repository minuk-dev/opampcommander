package usermodel

// PermissionAgentRead is the permission for reading agents.
const (
	PermissionAgentRead    = "agent:read"
	PermissionAgentWrite   = "agent:write"
	PermissionAgentDelete  = "agent:delete"
	PermissionAgentExecute = "agent:execute"

	// PermissionAgentGroupRead is the permission for reading agent groups.
	PermissionAgentGroupRead   = "agentgroup:read"
	PermissionAgentGroupWrite  = "agentgroup:write"
	PermissionAgentGroupDelete = "agentgroup:delete"

	// PermissionAgentPackageRead is the permission for reading agent packages.
	PermissionAgentPackageRead   = "agentpackage:read"
	PermissionAgentPackageWrite  = "agentpackage:write"
	PermissionAgentPackageDelete = "agentpackage:delete"

	// PermissionAgentRemoteConfigRead is the permission for reading agent remote configs.
	PermissionAgentRemoteConfigRead   = "agentremoteconfig:read"
	PermissionAgentRemoteConfigWrite  = "agentremoteconfig:write"
	PermissionAgentRemoteConfigDelete = "agentremoteconfig:delete"

	// PermissionCertificateRead is the permission for reading certificates.
	PermissionCertificateRead   = "certificate:read"
	PermissionCertificateWrite  = "certificate:write"
	PermissionCertificateDelete = "certificate:delete"

	// PermissionServerRead is the permission for reading server settings.
	PermissionServerRead   = "server:read"
	PermissionServerWrite  = "server:write"
	PermissionServerDelete = "server:delete"

	// PermissionUserRead is the permission for reading users.
	PermissionUserRead   = "user:read"
	PermissionUserWrite  = "user:write"
	PermissionUserDelete = "user:delete"

	// PermissionRoleRead is the permission for reading roles.
	PermissionRoleRead   = "role:read"
	PermissionRoleWrite  = "role:write"
	PermissionRoleDelete = "role:delete"

	// PermissionPermissionRead is the permission for reading permissions.
	PermissionPermissionRead   = "permission:read"
	PermissionPermissionWrite  = "permission:write"
	PermissionPermissionDelete = "permission:delete"
)

// RoleAdmin is the built-in admin role name.
const (
	RoleAdmin     = "Admin"
	RoleAgentMgr  = "AgentManager"
	RoleConfigMgr = "ConfigurationManager"
	RoleViewer    = "Viewer"
)

// ResourceAgent is the agent resource type.
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

// ActionRead is the read action.
const (
	ActionRead    = "read"
	ActionWrite   = "write"
	ActionDelete  = "delete"
	ActionExecute = "execute"
)
