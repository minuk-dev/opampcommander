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

// DefaultNamespace is the namespace used for built-in default role assignments.
const DefaultNamespace = "default"

// Built-in role names.
const (
	RoleSuperAdmin = "SuperAdmin"
	RoleAdmin      = "Admin"
	RoleViewer     = "Viewer"
	RoleDefault    = "default" // default role assigned to all new users; undeletable but permissions can be changed
)

// Label keys added to users on login.
const (
	LabelLoginType = "login-type"  // e.g. "github", "basic"
	LabelGitHubOrg = "github-org-" // prefix; full key = "github-org-{orgname}"
)

// NamespaceScopedResources returns all namespace-scoped resources controlled by RBAC.
func NamespaceScopedResources() []string {
	return []string{
		ResourceAgent,
		ResourceAgentGroup,
		ResourceAgentPackage,
		ResourceAgentRemoteConfig,
		ResourceCertificate,
		ResourceRoleBinding,
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
