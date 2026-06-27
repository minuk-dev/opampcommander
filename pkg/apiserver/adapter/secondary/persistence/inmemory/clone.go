package inmemory

import (
	"maps"
	"slices"
	"time"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	usermodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user"
)

// The clone* functions deep-copy a domain value so the in-memory store never
// shares mutable state (maps, slices, pointers) with its callers — reproducing
// the MongoDB adapter's fresh-copy-per-call semantics. Each one copies the
// top-level struct by value (handling all scalar fields) and then replaces every
// reference-typed field with an independent copy.
//
// Agent and Server already expose deep Clone() methods, so the store reuses them
// directly rather than duplicating the logic here.

func cloneTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}

	cloned := *t

	return &cloned
}

func cloneHost(host *agentmodel.Host) *agentmodel.Host {
	if host == nil {
		return nil
	}

	cloned := *host
	cloned.Metadata.Labels = maps.Clone(host.Metadata.Labels)
	cloned.Metadata.Annotations = maps.Clone(host.Metadata.Annotations)
	cloned.Status.AgentInstanceUIDs = slices.Clone(host.Status.AgentInstanceUIDs)
	cloned.Status.Conditions = slices.Clone(host.Status.Conditions)

	return &cloned
}

func cloneContainer(container *agentmodel.Container) *agentmodel.Container {
	if container == nil {
		return nil
	}

	cloned := *container
	cloned.Metadata.Labels = maps.Clone(container.Metadata.Labels)
	cloned.Metadata.Annotations = maps.Clone(container.Metadata.Annotations)
	cloned.Status.AgentInstanceUIDs = slices.Clone(container.Status.AgentInstanceUIDs)
	cloned.Status.Conditions = slices.Clone(container.Status.Conditions)

	return &cloned
}

func cloneStringPtr(str *string) *string {
	if str == nil {
		return nil
	}

	cloned := *str

	return &cloned
}

func cloneBytes(data []byte) []byte {
	if data == nil {
		return nil
	}

	return slices.Clone(data)
}

// cloneHeaders deep-copies a header map, including each value slice (maps.Clone
// alone would leave the []string values shared).
func cloneHeaders(headers map[string][]string) map[string][]string {
	if headers == nil {
		return nil
	}

	cloned := make(map[string][]string, len(headers))
	for key, values := range headers {
		cloned[key] = slices.Clone(values)
	}

	return cloned
}

func cloneNamespace(namespace *agentmodel.Namespace) *agentmodel.Namespace {
	if namespace == nil {
		return nil
	}

	cloned := *namespace
	cloned.Metadata.Labels = maps.Clone(namespace.Metadata.Labels)
	cloned.Metadata.Annotations = maps.Clone(namespace.Metadata.Annotations)
	cloned.Metadata.DeletedAt = cloneTimePtr(namespace.Metadata.DeletedAt)
	cloned.Status.Conditions = slices.Clone(namespace.Status.Conditions)

	return &cloned
}

func cloneAgentGroup(agentGroup *agentmodel.AgentGroup) *agentmodel.AgentGroup {
	if agentGroup == nil {
		return nil
	}

	cloned := *agentGroup
	cloned.Metadata.Attributes = maps.Clone(agentGroup.Metadata.Attributes)
	cloned.Spec.Selector.IdentifyingAttributes = maps.Clone(agentGroup.Spec.Selector.IdentifyingAttributes)
	cloned.Spec.Selector.NonIdentifyingAttributes = maps.Clone(agentGroup.Spec.Selector.NonIdentifyingAttributes)
	cloned.Spec.AgentConnectionConfig = cloneAgentGroupConnectionConfig(agentGroup.Spec.AgentConnectionConfig)
	cloned.Status.Conditions = slices.Clone(agentGroup.Status.Conditions)

	if agentGroup.Spec.AgentRemoteConfigs != nil {
		configs := make([]agentmodel.AgentGroupAgentRemoteConfig, len(agentGroup.Spec.AgentRemoteConfigs))
		for i := range agentGroup.Spec.AgentRemoteConfigs {
			configs[i] = *cloneAgentGroupRemoteConfig(&agentGroup.Spec.AgentRemoteConfigs[i])
		}

		cloned.Spec.AgentRemoteConfigs = configs
	}

	return &cloned
}

func cloneAgentGroupRemoteConfig(
	remoteConfig *agentmodel.AgentGroupAgentRemoteConfig,
) *agentmodel.AgentGroupAgentRemoteConfig {
	if remoteConfig == nil {
		return nil
	}

	cloned := *remoteConfig
	cloned.AgentRemoteConfigName = cloneStringPtr(remoteConfig.AgentRemoteConfigName)
	cloned.AgentRemoteConfigRef = cloneStringPtr(remoteConfig.AgentRemoteConfigRef)

	if remoteConfig.AgentRemoteConfigSpec != nil {
		spec := *remoteConfig.AgentRemoteConfigSpec
		spec.Value = cloneBytes(remoteConfig.AgentRemoteConfigSpec.Value)
		cloned.AgentRemoteConfigSpec = &spec
	}

	return &cloned
}

func cloneAgentGroupConnectionConfig(
	connectionConfig *agentmodel.AgentGroupConnectionConfig,
) *agentmodel.AgentGroupConnectionConfig {
	if connectionConfig == nil {
		return nil
	}

	cloned := *connectionConfig
	cloned.OpAMPConnection = cloneOpAMPConnectionSettings(connectionConfig.OpAMPConnection)
	cloned.OwnMetrics = cloneTelemetryConnectionSettings(connectionConfig.OwnMetrics)
	cloned.OwnLogs = cloneTelemetryConnectionSettings(connectionConfig.OwnLogs)
	cloned.OwnTraces = cloneTelemetryConnectionSettings(connectionConfig.OwnTraces)

	if connectionConfig.OtherConnections != nil {
		others := make(map[string]agentmodel.OtherConnectionSettings, len(connectionConfig.OtherConnections))
		for key, value := range connectionConfig.OtherConnections {
			others[key] = agentmodel.OtherConnectionSettings{
				DestinationEndpoint: value.DestinationEndpoint,
				Headers:             cloneHeaders(value.Headers),
				CertificateName:     cloneStringPtr(value.CertificateName),
			}
		}

		cloned.OtherConnections = others
	}

	return &cloned
}

func cloneOpAMPConnectionSettings(
	settings *agentmodel.OpAMPConnectionSettings,
) *agentmodel.OpAMPConnectionSettings {
	if settings == nil {
		return nil
	}

	return &agentmodel.OpAMPConnectionSettings{
		DestinationEndpoint: settings.DestinationEndpoint,
		Headers:             cloneHeaders(settings.Headers),
		CertificateName:     cloneStringPtr(settings.CertificateName),
	}
}

func cloneTelemetryConnectionSettings(
	settings *agentmodel.TelemetryConnectionSettings,
) *agentmodel.TelemetryConnectionSettings {
	if settings == nil {
		return nil
	}

	return &agentmodel.TelemetryConnectionSettings{
		DestinationEndpoint: settings.DestinationEndpoint,
		Headers:             cloneHeaders(settings.Headers),
		CertificateName:     cloneStringPtr(settings.CertificateName),
	}
}

func cloneAgentPackage(agentPackage *agentmodel.AgentPackage) *agentmodel.AgentPackage {
	if agentPackage == nil {
		return nil
	}

	cloned := *agentPackage
	cloned.Metadata.Attributes = maps.Clone(agentPackage.Metadata.Attributes)
	cloned.Metadata.DeletedAt = cloneTimePtr(agentPackage.Metadata.DeletedAt)
	cloned.Spec.ContentHash = cloneBytes(agentPackage.Spec.ContentHash)
	cloned.Spec.Signature = cloneBytes(agentPackage.Spec.Signature)
	cloned.Spec.Headers = maps.Clone(agentPackage.Spec.Headers)
	cloned.Spec.Hash = cloneBytes(agentPackage.Spec.Hash)
	cloned.Status.Conditions = slices.Clone(agentPackage.Status.Conditions)

	return &cloned
}

func cloneAgentRemoteConfig(remoteConfig *agentmodel.AgentRemoteConfig) *agentmodel.AgentRemoteConfig {
	if remoteConfig == nil {
		return nil
	}

	cloned := *remoteConfig
	cloned.Metadata.Attributes = maps.Clone(remoteConfig.Metadata.Attributes)
	cloned.Metadata.DeletedAt = cloneTimePtr(remoteConfig.Metadata.DeletedAt)
	cloned.Spec.Value = cloneBytes(remoteConfig.Spec.Value)
	cloned.Status.Conditions = slices.Clone(remoteConfig.Status.Conditions)

	return &cloned
}

func cloneEndpoint(endpoint *agentmodel.Endpoint) *agentmodel.Endpoint {
	if endpoint == nil {
		return nil
	}

	cloned := *endpoint
	cloned.Metadata.Attributes = maps.Clone(endpoint.Metadata.Attributes)
	cloned.Metadata.DeletedAt = cloneTimePtr(endpoint.Metadata.DeletedAt)
	cloned.Status.Conditions = slices.Clone(endpoint.Status.Conditions)

	if endpoint.Spec.MetricsQuery != nil {
		metricsQuery := *endpoint.Spec.MetricsQuery
		cloned.Spec.MetricsQuery = &metricsQuery
	}

	if endpoint.Spec.Tenants != nil {
		tenants := make([]agentmodel.EndpointTenant, len(endpoint.Spec.Tenants))
		for i := range endpoint.Spec.Tenants {
			tenants[i] = cloneEndpointTenant(endpoint.Spec.Tenants[i])
		}

		cloned.Spec.Tenants = tenants
	}

	return &cloned
}

func cloneEndpointTenant(tenant agentmodel.EndpointTenant) agentmodel.EndpointTenant {
	cloned := tenant
	cloned.Headers = maps.Clone(tenant.Headers)
	cloned.Tags = maps.Clone(tenant.Tags)

	if tenant.Signals != nil {
		signals := *tenant.Signals
		cloned.Signals = &signals
	}

	return cloned
}

func cloneCertificate(certificate *agentmodel.Certificate) *agentmodel.Certificate {
	if certificate == nil {
		return nil
	}

	cloned := *certificate
	cloned.Metadata.Attributes = maps.Clone(certificate.Metadata.Attributes)
	cloned.Spec.Cert = cloneBytes(certificate.Spec.Cert)
	cloned.Spec.PrivateKey = cloneBytes(certificate.Spec.PrivateKey)
	cloned.Spec.CaCert = cloneBytes(certificate.Spec.CaCert)
	cloned.Status.Conditions = slices.Clone(certificate.Status.Conditions)

	return &cloned
}

func cloneUser(user *usermodel.User) *usermodel.User {
	if user == nil {
		return nil
	}

	cloned := *user
	cloned.Metadata.DeletedAt = cloneTimePtr(user.Metadata.DeletedAt)
	cloned.Metadata.Labels = maps.Clone(user.Metadata.Labels)
	cloned.Spec.Identities = slices.Clone(user.Spec.Identities)
	cloned.Status.Conditions = slices.Clone(user.Status.Conditions)
	cloned.Status.Roles = slices.Clone(user.Status.Roles)

	if user.Spec.BasicAuth != nil {
		basicAuth := *user.Spec.BasicAuth
		cloned.Spec.BasicAuth = &basicAuth
	}

	return &cloned
}

func cloneRole(role *usermodel.Role) *usermodel.Role {
	if role == nil {
		return nil
	}

	cloned := *role
	cloned.Metadata.DeletedAt = cloneTimePtr(role.Metadata.DeletedAt)
	cloned.Spec.Permissions = slices.Clone(role.Spec.Permissions)
	cloned.Status.Conditions = slices.Clone(role.Status.Conditions)

	return &cloned
}

func clonePermission(permission *usermodel.Permission) *usermodel.Permission {
	if permission == nil {
		return nil
	}

	cloned := *permission
	cloned.Metadata.DeletedAt = cloneTimePtr(permission.Metadata.DeletedAt)
	cloned.Status.Conditions = slices.Clone(permission.Status.Conditions)

	return &cloned
}

func cloneUserRole(userRole *usermodel.UserRole) *usermodel.UserRole {
	if userRole == nil {
		return nil
	}

	cloned := *userRole
	cloned.Metadata.DeletedAt = cloneTimePtr(userRole.Metadata.DeletedAt)
	cloned.Status.Conditions = slices.Clone(userRole.Status.Conditions)

	return &cloned
}

func cloneRoleBinding(roleBinding *usermodel.RoleBinding) *usermodel.RoleBinding {
	if roleBinding == nil {
		return nil
	}

	cloned := *roleBinding
	cloned.Metadata.DeletedAt = cloneTimePtr(roleBinding.Metadata.DeletedAt)
	cloned.Spec.Subjects = slices.Clone(roleBinding.Spec.Subjects)
	cloned.Status.Conditions = slices.Clone(roleBinding.Status.Conditions)

	return &cloned
}
