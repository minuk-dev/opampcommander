package usermodel

// WildcardAll represents a wildcard matching all values in RBAC policies.
const WildcardAll = "*"

// Actions for RBAC permission checks.
const (
	ActionGet    = "GET"
	ActionList   = "LIST"
	ActionCreate = "CREATE"
	ActionUpdate = "UPDATE"
	ActionDelete = "DELETE"
)

// Namespace-scoped resource types controlled by RBAC.
const (
	ResourceAgent             = "agent"
	ResourceAgentGroup        = "agentgroup"
	ResourceAgentPackage      = "agentpackage"
	ResourceAgentRemoteConfig = "agentremoteconfig"
	ResourceCertificate       = "certificate"
	ResourceRoleBinding       = "rolebinding"
)

// Global resource types (not namespace-scoped).
const (
	ResourceServer     = "server"
	ResourceUser       = "user"
	ResourceRole       = "role"
	ResourcePermission = "permission"
)

// Built-in role names.
const (
	RoleSuperAdmin = "SuperAdmin"
	RoleAdmin      = "Admin"
	RoleViewer     = "Viewer"
)

// NamespaceScopedResources returns all namespace-scoped resources controlled by RBAC.
func NamespaceScopedResources() []string {
	return []string{
		ResourceAgent,
		ResourceAgentGroup,
		ResourceAgentPackage,
		ResourceAgentRemoteConfig,
		ResourceCertificate,
	}
}

// AllActions returns all RBAC actions.
func AllActions() []string {
	return []string{ActionGet, ActionList, ActionCreate, ActionUpdate, ActionDelete}
}

// ReadOnlyActions returns read-only RBAC actions.
func ReadOnlyActions() []string {
	return []string{ActionGet, ActionList}
}
