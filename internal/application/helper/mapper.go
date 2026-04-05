// Package helper provides helper functions for mapping between domain models and API models.
package helper

import (
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/agent/model"
	"github.com/minuk-dev/opampcommander/internal/domain/agent/model/agent"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	usermodel "github.com/minuk-dev/opampcommander/internal/domain/user/model"
)

// Mapper is a struct that provides methods to map between domain models and API models.
type Mapper struct{}

// NewMapper creates a new instance of Mapper.
func NewMapper() *Mapper {
	return &Mapper{}
}

// MapAPIToAgentGroup maps an API model AgentGroup to a domain model AgentGroup.
func (mapper *Mapper) MapAPIToAgentGroup(apiAgentGroup *v1.AgentGroup) *agentmodel.AgentGroup {
	if apiAgentGroup == nil {
		return nil
	}

	var agentRemoteConfig *agentmodel.AgentGroupAgentRemoteConfig

	if apiAgentGroup.Spec.AgentConfig != nil && apiAgentGroup.Spec.AgentConfig.AgentRemoteConfig != nil {
		apiRemoteConfig := apiAgentGroup.Spec.AgentConfig.AgentRemoteConfig

		agentRemoteConfig = &agentmodel.AgentGroupAgentRemoteConfig{
			AgentRemoteConfigName: apiRemoteConfig.AgentRemoteConfigName,
			AgentRemoteConfigRef:  apiRemoteConfig.AgentRemoteConfigRef,
			AgentRemoteConfigSpec: nil,
		}
		if apiRemoteConfig.AgentRemoteConfigSpec != nil {
			agentRemoteConfig.AgentRemoteConfigSpec = &agentmodel.AgentRemoteConfigSpec{
				Value:       []byte(apiRemoteConfig.AgentRemoteConfigSpec.Value),
				ContentType: apiRemoteConfig.AgentRemoteConfigSpec.ContentType,
			}
		}
	}

	var agentConnectionConfig *agentmodel.AgentGroupConnectionConfig

	if apiAgentGroup.Spec.AgentConfig != nil && apiAgentGroup.Spec.AgentConfig.ConnectionSettings != nil {
		connSettings := apiAgentGroup.Spec.AgentConfig.ConnectionSettings
		agentConnectionConfig = &agentmodel.AgentGroupConnectionConfig{
			OpAMPConnection:  mapper.mapOpAMPConnectionFromAPI(connSettings.OpAMP),
			OwnMetrics:       mapper.mapTelemetryConnectionFromAPI(connSettings.OwnMetrics),
			OwnLogs:          mapper.mapTelemetryConnectionFromAPI(connSettings.OwnLogs),
			OwnTraces:        mapper.mapTelemetryConnectionFromAPI(connSettings.OwnTraces),
			OtherConnections: mapper.mapOtherConnectionsFromAPI(connSettings.OtherConnections),
		}
	}

	//exhaustruct:ignore
	return &agentmodel.AgentGroup{
		Metadata: agentmodel.AgentGroupMetadata{
			Namespace:  apiAgentGroup.Metadata.Namespace,
			Name:       apiAgentGroup.Metadata.Name,
			Attributes: agentmodel.OfAttributes(apiAgentGroup.Metadata.Attributes),
		},
		Spec: agentmodel.AgentGroupSpec{
			Priority: apiAgentGroup.Spec.Priority,
			Selector: agentmodel.AgentSelector{
				IdentifyingAttributes:    apiAgentGroup.Spec.Selector.IdentifyingAttributes,
				NonIdentifyingAttributes: apiAgentGroup.Spec.Selector.NonIdentifyingAttributes,
			},
			AgentRemoteConfig:     agentRemoteConfig,
			AgentConnectionConfig: agentConnectionConfig,
		},
		// Note: Status is not mapped here as it is usually managed by the system.
	}
}

// MapAgentGroupToAPI maps a domain model AgentGroup to an API model AgentGroup.
func (mapper *Mapper) MapAgentGroupToAPI(domainAgentGroup *agentmodel.AgentGroup) *v1.AgentGroup {
	if domainAgentGroup == nil {
		return nil
	}

	var agentConfig *v1.AgentConfig

	//nolint:staticcheck // backward compatibility - AgentRemoteConfig is deprecated
	if domainAgentGroup.Spec.AgentRemoteConfig != nil || domainAgentGroup.Spec.AgentConnectionConfig != nil {
		//exhaustruct:ignore
		agentConfig = &v1.AgentConfig{}

		//nolint:staticcheck // backward compatibility - AgentRemoteConfig is deprecated
		if domainAgentGroup.Spec.AgentRemoteConfig != nil {
			agentConfig.AgentRemoteConfig = &v1.AgentGroupRemoteConfig{
				//nolint:staticcheck // backward compat
				AgentRemoteConfigName: domainAgentGroup.Spec.AgentRemoteConfig.AgentRemoteConfigName,
				//nolint:staticcheck // backward compat
				AgentRemoteConfigRef:  domainAgentGroup.Spec.AgentRemoteConfig.AgentRemoteConfigRef,
				AgentRemoteConfigSpec: nil,
			}

			//nolint:staticcheck // backward compatibility
			if domainAgentGroup.Spec.AgentRemoteConfig.AgentRemoteConfigSpec != nil {
				agentConfig.AgentRemoteConfig.AgentRemoteConfigSpec = &v1.AgentRemoteConfigSpec{
					//nolint:staticcheck // backward compat
					Value: string(domainAgentGroup.Spec.AgentRemoteConfig.AgentRemoteConfigSpec.Value),
					//nolint:staticcheck // backward compat
					ContentType: domainAgentGroup.Spec.AgentRemoteConfig.AgentRemoteConfigSpec.ContentType,
				}
			}
		}

		if domainAgentGroup.Spec.AgentConnectionConfig != nil {
			agentConfig.ConnectionSettings = &v1.ConnectionSettings{
				OpAMP:            mapper.mapOpAMPConnectionToAPI(domainAgentGroup.Spec.AgentConnectionConfig.OpAMPConnection),
				OwnMetrics:       mapper.mapTelemetryConnectionToAPI(domainAgentGroup.Spec.AgentConnectionConfig.OwnMetrics),
				OwnLogs:          mapper.mapTelemetryConnectionToAPI(domainAgentGroup.Spec.AgentConnectionConfig.OwnLogs),
				OwnTraces:        mapper.mapTelemetryConnectionToAPI(domainAgentGroup.Spec.AgentConnectionConfig.OwnTraces),
				OtherConnections: mapper.mapOtherConnectionsToAPI(domainAgentGroup.Spec.AgentConnectionConfig.OtherConnections),
			}
		}
	}

	return &v1.AgentGroup{
		Metadata: v1.Metadata{
			Namespace:  domainAgentGroup.Metadata.Namespace,
			Name:       domainAgentGroup.Metadata.Name,
			CreatedAt:  v1.NewTime(domainAgentGroup.Metadata.CreatedAt),
			DeletedAt:  p(v1.NewTime(domainAgentGroup.Metadata.DeletedAt)),
			Attributes: v1.Attributes(domainAgentGroup.Metadata.Attributes),
		},
		Spec: v1.Spec{
			Priority: domainAgentGroup.Spec.Priority,
			Selector: v1.AgentSelector{
				IdentifyingAttributes:    domainAgentGroup.Spec.Selector.IdentifyingAttributes,
				NonIdentifyingAttributes: domainAgentGroup.Spec.Selector.NonIdentifyingAttributes,
			},
			AgentConfig: agentConfig,
		},
		Status: v1.Status{
			NumAgents:             domainAgentGroup.Status.NumAgents,
			NumConnectedAgents:    domainAgentGroup.Status.NumConnectedAgents,
			NumHealthyAgents:      domainAgentGroup.Status.NumHealthyAgents,
			NumUnhealthyAgents:    domainAgentGroup.Status.NumUnhealthyAgents,
			NumNotConnectedAgents: domainAgentGroup.Status.NumNotConnectedAgents,
			Conditions:            mapper.mapAgentGroupConditionsToAPI(domainAgentGroup.Status.Conditions),
		},
	}
}

// MapAPIToAgent maps an API model Agent to a domain model Agent.
func (mapper *Mapper) MapAPIToAgent(apiAgent *v1.Agent) *agentmodel.Agent {
	//exhaustruct:ignore
	return &agentmodel.Agent{
		Metadata: agentmodel.AgentMetadata{
			InstanceUID: apiAgent.Metadata.InstanceUID,
			Namespace:   apiAgent.Metadata.Namespace,
			Description: agent.Description{
				IdentifyingAttributes:    apiAgent.Metadata.Description.IdentifyingAttributes,
				NonIdentifyingAttributes: apiAgent.Metadata.Description.NonIdentifyingAttributes,
			},
			Capabilities:       agent.Capabilities(apiAgent.Metadata.Capabilities),
			CustomCapabilities: mapper.mapCustomCapabilitiesFromAPI(&apiAgent.Metadata.CustomCapabilities),
		},
		//exhaustruct:ignore
		Spec: agentmodel.AgentSpec{
			NewInstanceUID:    mapper.mapNewInstanceUIDFromAPI(apiAgent.Spec.NewInstanceUID),
			RemoteConfig:      mapper.mapRemoteConfigFromAPI(&apiAgent.Spec.RemoteConfig),
			PackagesAvailable: mapper.mapPackagesAvailableFromAPI(&apiAgent.Spec.PackagesAvailable),
			RestartInfo:       mapper.mapRestartInfoFromAPI(apiAgent.Spec.RestartRequiredAt),
		},
		// Note: Status is not mapped here as it is usually managed by the system.
	}
}

// MapAgentToAPI maps a domain model Agent to an API model Agent.
func (mapper *Mapper) MapAgentToAPI(agent *agentmodel.Agent) *v1.Agent {
	return &v1.Agent{
		Metadata: v1.AgentMetadata{
			InstanceUID: agent.Metadata.InstanceUID,
			Namespace:   agent.Metadata.Namespace,
			Description: v1.AgentDescription{
				IdentifyingAttributes:    agent.Metadata.Description.IdentifyingAttributes,
				NonIdentifyingAttributes: agent.Metadata.Description.NonIdentifyingAttributes,
			},
			Capabilities:       v1.AgentCapabilities(agent.Metadata.Capabilities),
			CustomCapabilities: mapper.mapCustomCapabilitiesToAPI(&agent.Metadata.CustomCapabilities),
		},
		//exhaustruct:ignore
		Spec: v1.AgentSpec{
			NewInstanceUID:    mapper.mapNewInstanceUIDToAPI(agent.Spec.NewInstanceUID[:]),
			RemoteConfig:      mapper.mapRemoteConfigToAPI(agent.Spec.RemoteConfig),
			PackagesAvailable: mapper.mapPackagesAvailableToAPI(agent.Spec.PackagesAvailable),
			RestartRequiredAt: mapper.mapRestartRequiredAtToAPI(agent.Spec.RestartInfo),
		},
		Status: v1.AgentStatus{
			EffectiveConfig: v1.AgentEffectiveConfig{
				ConfigMap: v1.AgentConfigMap{
					ConfigMap: lo.MapValues(agent.Status.EffectiveConfig.ConfigMap.ConfigMap,
						func(value agentmodel.AgentConfigFile, _ string) v1.AgentConfigFile {
							return mapper.mapConfigFileToAPI(value)
						}),
				},
			},
			PackageStatuses: v1.AgentPackageStatuses{
				Packages: lo.MapValues(agent.Status.PackageStatuses.Packages,
					func(value agentmodel.AgentPackageStatusEntry, _ string) v1.AgentStatusPackageEntry {
						return v1.AgentStatusPackageEntry{
							Name: value.Name,
						}
					}),
				ServerProvidedAllPackagesHash: string(agent.Status.PackageStatuses.ServerProvidedAllPackgesHash),
				ErrorMessage:                  agent.Status.PackageStatuses.ErrorMessage,
			},
			ComponentHealth:     mapper.mapComponentHealthToAPI(&agent.Status.ComponentHealth),
			AvailableComponents: mapper.mapAvailableComponentsToAPI(&agent.Status.AvailableComponents),
			Conditions:          mapper.mapAgentConditionsToAPI(agent.Status.Conditions),
			Connected:           agent.Status.Connected,
			ConnectionType:      agent.Status.ConnectionType.String(),
			SequenceNum:         agent.Status.SequenceNum,
			LastReportedAt:      mapper.formatTime(agent.Status.LastReportedAt),
		},
	}
}

// MapAgentPackageToAPI maps a domain model AgentPackage to an API model AgentPackage.
func (mapper *Mapper) MapAgentPackageToAPI(agentPackage *agentmodel.AgentPackage) *v1.AgentPackage {
	var deletedAt *v1.Time

	if agentPackage.Metadata.DeletedAt != nil {
		t := v1.NewTime(*agentPackage.Metadata.DeletedAt)
		deletedAt = &t
	}

	return &v1.AgentPackage{
		Metadata: v1.AgentPackageMetadata{
			Name:       agentPackage.Metadata.Name,
			Namespace:  agentPackage.Metadata.Namespace,
			Attributes: v1.Attributes(agentPackage.Metadata.Attributes),
			CreatedAt:  v1.NewTime(agentPackage.Metadata.CreatedAt),
			DeletedAt:  deletedAt,
		},
		Spec: v1.AgentPackageSpec{
			PackageType: agentPackage.Spec.PackageType,
			Version:     agentPackage.Spec.Version,
			DownloadURL: agentPackage.Spec.DownloadURL,
			ContentHash: agentPackage.Spec.ContentHash,
			Signature:   agentPackage.Spec.Signature,
			Headers:     agentPackage.Spec.Headers,
			Hash:        agentPackage.Spec.Hash,
		},
		Status: v1.AgentPackageStatus{
			Conditions: mapper.mapConditionsToAPI(agentPackage.Status.Conditions),
		},
	}
}

// MapAPIToAgentPackage maps an API model AgentPackage to a domain model AgentPackage.
func (mapper *Mapper) MapAPIToAgentPackage(apiModel *v1.AgentPackage) *agentmodel.AgentPackage {
	return &agentmodel.AgentPackage{
		Metadata: agentmodel.AgentPackageMetadata{
			Name:       apiModel.Metadata.Name,
			Namespace:  apiModel.Metadata.Namespace,
			Attributes: agentmodel.OfAttributes(apiModel.Metadata.Attributes),
			CreatedAt:  apiModel.Metadata.CreatedAt.Time,
			DeletedAt:  nil,
		},
		Spec: agentmodel.AgentPackageSpec{
			PackageType: apiModel.Spec.PackageType,
			Version:     apiModel.Spec.Version,
			DownloadURL: apiModel.Spec.DownloadURL,
			ContentHash: apiModel.Spec.ContentHash,
			Signature:   apiModel.Spec.Signature,
			Headers:     apiModel.Spec.Headers,
			Hash:        apiModel.Spec.Hash,
		},
		Status: agentmodel.AgentPackageStatus{
			// Skip mapping conditions as they are usually managed by the system.
			Conditions: nil,
		},
	}
}

// MapCertificateToAPI maps a domain model Certificate to an API model Certificate.
func (mapper *Mapper) MapCertificateToAPI(domain *agentmodel.Certificate) *v1.Certificate {
	if domain == nil {
		return nil
	}

	return &v1.Certificate{
		Kind:       v1.CertificateKind,
		APIVersion: v1.APIVersion,
		Metadata: v1.CertificateMetadata{
			Name:       domain.Metadata.Name,
			Namespace:  domain.Metadata.Namespace,
			Attributes: v1.Attributes(domain.Metadata.Attributes),
			CreatedAt:  v1.NewTime(domain.Metadata.CreatedAt),
			DeletedAt:  p(v1.NewTime(domain.Metadata.DeletedAt)),
		},
		Spec: v1.CertificateSpec{
			Cert:       string(domain.Spec.Cert),
			PrivateKey: string(domain.Spec.PrivateKey),
			CaCert:     string(domain.Spec.CaCert),
		},
		Status: v1.CertificateStatus{
			Conditions: mapper.mapAgentGroupConditionsToAPI(domain.Status.Conditions),
		},
	}
}

// MapAPIToCertificate maps an API model Certificate to a domain model Certificate.
func (mapper *Mapper) MapAPIToCertificate(api *v1.Certificate) *agentmodel.Certificate {
	if api == nil {
		return nil
	}

	var deletedAt time.Time
	if api.Metadata.DeletedAt != nil {
		deletedAt = api.Metadata.DeletedAt.Time
	}

	return &agentmodel.Certificate{
		Metadata: agentmodel.CertificateMetadata{
			Name:       api.Metadata.Name,
			Namespace:  api.Metadata.Namespace,
			Attributes: agentmodel.Attributes(api.Metadata.Attributes),
			CreatedAt:  api.Metadata.CreatedAt.Time,
			DeletedAt:  deletedAt,
		},
		Spec: agentmodel.CertificateSpec{
			Cert:       []byte(api.Spec.Cert),
			PrivateKey: []byte(api.Spec.PrivateKey),
			CaCert:     []byte(api.Spec.CaCert),
		},
		Status: agentmodel.CertificateStatus{
			Conditions: nil,
		},
	}
}

// MapAgentRemoteConfigToAPI maps a domain model AgentRemoteConfig to an API model.
func (mapper *Mapper) MapAgentRemoteConfigToAPI(
	domain *agentmodel.AgentRemoteConfig,
) *v1.AgentRemoteConfig {
	if domain == nil {
		return nil
	}

	return &v1.AgentRemoteConfig{
		Metadata: v1.AgentRemoteConfigMetadata{
			Name:       domain.Metadata.Name,
			Namespace:  domain.Metadata.Namespace,
			Attributes: v1.Attributes(domain.Metadata.Attributes),
			CreatedAt:  v1.NewTime(domain.Metadata.CreatedAt),
		},
		Spec: v1.AgentRemoteConfigSpec{
			Value:       string(domain.Spec.Value),
			ContentType: domain.Spec.ContentType,
		},
		Status: v1.AgentRemoteConfigStatus{
			Conditions: mapper.mapConditionsToAPI(domain.Status.Conditions),
		},
	}
}

// MapAPIToAgentRemoteConfig maps an API model AgentRemoteConfig to a domain model.
func (mapper *Mapper) MapAPIToAgentRemoteConfig(
	api *v1.AgentRemoteConfig,
) *agentmodel.AgentRemoteConfig {
	if api == nil {
		return nil
	}

	return &agentmodel.AgentRemoteConfig{
		Metadata: agentmodel.AgentRemoteConfigMetadata{
			Name:       api.Metadata.Name,
			Namespace:  api.Metadata.Namespace,
			Attributes: agentmodel.OfAttributes(api.Metadata.Attributes),
			CreatedAt:  api.Metadata.CreatedAt.Time,
			DeletedAt:  nil,
		},
		Spec: agentmodel.AgentRemoteConfigSpec{
			Value:       []byte(api.Spec.Value),
			ContentType: api.Spec.ContentType,
		},
		Status: agentmodel.AgentRemoteConfigResourceStatus{
			Conditions: nil,
		},
	}
}

// MapUserToAPI maps a domain model User to an API model User.
func (mapper *Mapper) MapUserToAPI(domain *usermodel.User) *v1.User {
	if domain == nil {
		return nil
	}

	return &v1.User{
		Kind:       v1.UserKind,
		APIVersion: v1.APIVersion,
		Metadata: v1.UserMetadata{
			UID:       domain.Metadata.UID.String(),
			CreatedAt: v1.NewTime(domain.Metadata.CreatedAt),
			UpdatedAt: v1.NewTime(domain.Metadata.UpdatedAt),
			DeletedAt: mapDeletedAtPtrToAPI(domain.Metadata.DeletedAt),
		},
		Spec: v1.UserSpec{
			Email:    domain.Spec.Email,
			Username: domain.Spec.Username,
			IsActive: domain.Spec.IsActive,
		},
		Status: v1.UserStatus{
			Conditions: mapper.mapConditionsToAPI(domain.Status.Conditions),
			Roles:      domain.Status.Roles,
		},
	}
}

// MapRoleToAPI maps a domain model Role to an API model Role.
func (mapper *Mapper) MapRoleToAPI(domain *usermodel.Role) *v1.Role {
	if domain == nil {
		return nil
	}

	return &v1.Role{
		Kind:       v1.RoleKind,
		APIVersion: v1.APIVersion,
		Metadata: v1.RoleMetadata{
			UID:       domain.Metadata.UID.String(),
			CreatedAt: v1.NewTime(domain.Metadata.CreatedAt),
			UpdatedAt: v1.NewTime(domain.Metadata.UpdatedAt),
			DeletedAt: mapDeletedAtPtrToAPI(domain.Metadata.DeletedAt),
		},
		Spec: v1.RoleSpec{
			DisplayName: domain.Spec.DisplayName,
			Description: domain.Spec.Description,
			Permissions: domain.Spec.Permissions,
			IsBuiltIn:   domain.Spec.IsBuiltIn,
		},
		Status: v1.RoleStatus{
			Conditions: mapper.mapConditionsToAPI(domain.Status.Conditions),
		},
	}
}

// MapPermissionToAPI maps a domain model Permission to an API model Permission.
func (mapper *Mapper) MapPermissionToAPI(domain *usermodel.Permission) *v1.Permission {
	if domain == nil {
		return nil
	}

	return &v1.Permission{
		Kind:       v1.PermissionKind,
		APIVersion: v1.APIVersion,
		Metadata: v1.PermissionMetadata{
			UID:       domain.Metadata.UID.String(),
			CreatedAt: v1.NewTime(domain.Metadata.CreatedAt),
			UpdatedAt: v1.NewTime(domain.Metadata.UpdatedAt),
			DeletedAt: mapDeletedAtPtrToAPI(domain.Metadata.DeletedAt),
		},
		Spec: v1.PermissionSpec{
			Name:        domain.Spec.Name,
			Description: domain.Spec.Description,
			Resource:    domain.Spec.Resource,
			Action:      domain.Spec.Action,
			IsBuiltIn:   domain.Spec.IsBuiltIn,
		},
		Status: v1.PermissionStatus{
			Conditions: mapper.mapConditionsToAPI(domain.Status.Conditions),
		},
	}
}

// MapUserRoleToAPI maps a domain model UserRole to an API model UserRole.
func (mapper *Mapper) MapUserRoleToAPI(domain *usermodel.UserRole) *v1.UserRole {
	if domain == nil {
		return nil
	}

	return &v1.UserRole{
		Kind:       v1.UserRoleKind,
		APIVersion: v1.APIVersion,
		Metadata: v1.UserRoleMetadata{
			UID:       domain.Metadata.UID.String(),
			CreatedAt: v1.NewTime(domain.Metadata.CreatedAt),
			UpdatedAt: v1.NewTime(domain.Metadata.UpdatedAt),
			DeletedAt: mapDeletedAtPtrToAPI(domain.Metadata.DeletedAt),
		},
		Spec: v1.UserRoleSpec{
			UserID:     domain.Spec.UserID.String(),
			RoleID:     domain.Spec.RoleID.String(),
			AssignedAt: v1.NewTime(domain.Spec.AssignedAt),
			AssignedBy: domain.Spec.AssignedBy.String(),
		},
		Status: v1.UserRoleStatus{
			Conditions: mapper.mapConditionsToAPI(domain.Status.Conditions),
		},
	}
}

// --------------------------------------------------------------------------
// Private helper methods (placed after all exported methods per funcorder)
// --------------------------------------------------------------------------

const (
	// TextJSON is the content type for JSON.
	TextJSON = "text/json"
	// TextYAML is the content type for YAML.
	TextYAML = "text/yaml"
	// Empty is the content type for empty.
	// Empty content type is treated as YAML by default.
	// Due to spec miss, old otel-collector sends empty content type even though it should be YAML.
	Empty = ""
)

// MapNamespaceToAPI maps a domain Namespace to an API Namespace.
func (mapper *Mapper) MapNamespaceToAPI(
	namespace *agentmodel.Namespace,
) *v1.Namespace {
	return &v1.Namespace{
		Metadata: v1.NamespaceMetadata{
			Name:        namespace.Metadata.Name,
			Labels:      namespace.Metadata.Labels,
			Annotations: namespace.Metadata.Annotations,
			CreatedAt:   v1.NewTime(namespace.Metadata.CreatedAt),
			DeletedAt:   mapDeletedAtPtrToAPI(namespace.Metadata.DeletedAt),
		},
		Status: v1.NamespaceStatus{
			Conditions: mapper.mapConditionsToAPI(
				namespace.Status.Conditions,
			),
		},
	}
}

// MapAPIToNamespace maps an API Namespace to a domain Namespace.
func (mapper *Mapper) MapAPIToNamespace(
	apiModel *v1.Namespace,
) *agentmodel.Namespace {
	return &agentmodel.Namespace{
		Metadata: agentmodel.NamespaceMetadata{
			Name:        apiModel.Metadata.Name,
			Labels:      apiModel.Metadata.Labels,
			Annotations: apiModel.Metadata.Annotations,
			CreatedAt:   apiModel.Metadata.CreatedAt.Time,
			DeletedAt:   nil,
		},
		Status: agentmodel.NamespaceStatus{
			Conditions: nil,
		},
	}
}

func (mapper *Mapper) mapCustomCapabilitiesFromAPI(
	apiCustomCapabilities *v1.AgentCustomCapabilities,
) agentmodel.AgentCustomCapabilities {
	return agentmodel.AgentCustomCapabilities{
		Capabilities: apiCustomCapabilities.Capabilities,
	}
}

func (mapper *Mapper) mapRemoteConfigFromAPI(
	apiRemoteConfig *v1.AgentSpecRemoteConfig,
) *agentmodel.AgentSpecRemoteConfig {
	if apiRemoteConfig == nil || len(apiRemoteConfig.RemoteConfigNames) == 0 {
		return nil
	}

	// Convert RemoteConfigNames to ConfigMap with empty placeholders
	// The actual config values will be fetched when needed
	configMap := make(map[string]agentmodel.AgentConfigFile)
	for _, name := range apiRemoteConfig.RemoteConfigNames {
		configMap[name] = agentmodel.AgentConfigFile{
			Body:        nil,
			ContentType: "",
		}
	}

	return &agentmodel.AgentSpecRemoteConfig{
		ConfigMap: agentmodel.AgentConfigMap{
			ConfigMap: configMap,
		},
	}
}

func (mapper *Mapper) mapNewInstanceUIDFromAPI(newInstanceUID string) uuid.UUID {
	if newInstanceUID == "" {
		return uuid.Nil
	}

	uid, err := uuid.Parse(newInstanceUID)
	if err != nil {
		return uuid.Nil
	}

	return uid
}

func (mapper *Mapper) mapRestartInfoFromAPI(restartRequiredAt *v1.Time) *agentmodel.AgentRestartInfo {
	if restartRequiredAt == nil || restartRequiredAt.IsZero() {
		return nil
	}

	return &agentmodel.AgentRestartInfo{
		RequiredRestartedAt: restartRequiredAt.Time,
	}
}

func (mapper *Mapper) formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.Format(time.RFC3339)
}

func (mapper *Mapper) mapComponentHealthToAPI(health *agentmodel.AgentComponentHealth) v1.AgentComponentHealth {
	componentsMap := make(map[string]string)
	for name, comp := range health.ComponentHealthMap {
		componentsMap[name] = comp.Status
	}

	return v1.AgentComponentHealth{
		Healthy:       health.Healthy,
		StartTime:     v1.NewTime(health.StartTime),
		LastError:     health.LastError,
		Status:        health.Status,
		StatusTime:    v1.NewTime(health.StatusTime),
		ComponentsMap: componentsMap,
	}
}

func (mapper *Mapper) mapRemoteConfigToAPI(remoteConfig *agentmodel.AgentSpecRemoteConfig) v1.AgentSpecRemoteConfig {
	if remoteConfig == nil || len(remoteConfig.ConfigMap.ConfigMap) == 0 {
		//exhaustruct:ignore
		return v1.AgentSpecRemoteConfig{}
	}

	// Extract config names from ConfigMap
	names := make([]string, 0, len(remoteConfig.ConfigMap.ConfigMap))
	for name := range remoteConfig.ConfigMap.ConfigMap {
		names = append(names, name)
	}

	return v1.AgentSpecRemoteConfig{
		RemoteConfigNames: names,
	}
}

func (mapper *Mapper) mapPackagesAvailableToAPI(
	packagesAvailable *agentmodel.AgentSpecPackage,
) v1.AgentSpecPackages {
	if packagesAvailable == nil {
		//exhaustruct:ignore
		return v1.AgentSpecPackages{}
	}

	return v1.AgentSpecPackages{
		Packages: packagesAvailable.Packages,
	}
}

func (mapper *Mapper) mapRestartRequiredAtToAPI(restartInfo *agentmodel.AgentRestartInfo) *v1.Time {
	if restartInfo == nil || restartInfo.RequiredRestartedAt.IsZero() {
		return nil
	}

	t := v1.NewTime(restartInfo.RequiredRestartedAt)

	return &t
}

func (mapper *Mapper) mapCustomCapabilitiesToAPI(
	customCapabilities *agentmodel.AgentCustomCapabilities,
) v1.AgentCustomCapabilities {
	return v1.AgentCustomCapabilities{
		Capabilities: customCapabilities.Capabilities,
	}
}

func (mapper *Mapper) mapPackagesAvailableFromAPI(
	apiPackages *v1.AgentSpecPackages,
) *agentmodel.AgentSpecPackage {
	if apiPackages == nil {
		return nil
	}

	return &agentmodel.AgentSpecPackage{
		Packages: apiPackages.Packages,
	}
}

func (mapper *Mapper) mapAvailableComponentsToAPI(
	availableComponents *agentmodel.AgentAvailableComponents,
) v1.AgentAvailableComponents {
	components := lo.MapValues(availableComponents.Components,
		func(value agentmodel.ComponentDetails, _ string) v1.AgentComponentDetails {
			// Extract type and version from metadata if available
			componentType := value.Metadata["type"]
			version := value.Metadata["version"]

			return v1.AgentComponentDetails{
				Type:    componentType,
				Version: version,
			}
		})

	return v1.AgentAvailableComponents{
		Components: components,
	}
}

func (mapper *Mapper) mapConfigFileToAPI(configFile agentmodel.AgentConfigFile) v1.AgentConfigFile {
	switch configFile.ContentType {
	case TextJSON,
		TextYAML,
		Empty:
		return v1.AgentConfigFile{
			Body:        string(configFile.Body),
			ContentType: configFile.ContentType,
		}
	default:
		return v1.AgentConfigFile{
			Body:        "unsupported content type",
			ContentType: configFile.ContentType,
		}
	}
}

func (mapper *Mapper) mapAgentConditionsToAPI(conditions []agentmodel.AgentCondition) []v1.Condition {
	return lo.Map(conditions, func(condition agentmodel.AgentCondition, _ int) v1.Condition {
		return v1.Condition{
			Type:               v1.ConditionType(condition.Type),
			LastTransitionTime: v1.NewTime(condition.LastTransitionTime),
			Status:             v1.ConditionStatus(condition.Status),
			Reason:             condition.Reason,
			Message:            condition.Message,
		}
	})
}

func (mapper *Mapper) mapConditionsToAPI(conditions []model.Condition) []v1.Condition {
	return lo.Map(conditions, func(condition model.Condition, _ int) v1.Condition {
		return v1.Condition{
			Type:               v1.ConditionType(condition.Type),
			LastTransitionTime: v1.NewTime(condition.LastTransitionTime),
			Status:             v1.ConditionStatus(condition.Status),
			Reason:             condition.Reason,
			Message:            condition.Message,
		}
	})
}

func (mapper *Mapper) mapNewInstanceUIDToAPI(newInstanceUID []byte) string {
	if len(newInstanceUID) == 0 {
		return ""
	}

	return string(newInstanceUID)
}

func (mapper *Mapper) mapOpAMPConnectionFromAPI(
	apiConn v1.OpAMPConnectionSettings,
) *agentmodel.OpAMPConnectionSettings {
	return &agentmodel.OpAMPConnectionSettings{
		DestinationEndpoint: apiConn.DestinationEndpoint,
		Headers:             apiConn.Headers,
		CertificateName:     apiConn.CertificateName,
	}
}

func (mapper *Mapper) mapTelemetryConnectionFromAPI(
	apiConn v1.TelemetryConnectionSettings,
) *agentmodel.TelemetryConnectionSettings {
	return &agentmodel.TelemetryConnectionSettings{
		DestinationEndpoint: apiConn.DestinationEndpoint,
		Headers:             apiConn.Headers,
		CertificateName:     apiConn.CertificateName,
	}
}

func (mapper *Mapper) mapOtherConnectionsFromAPI(
	apiConns map[string]v1.OtherConnectionSettings,
) map[string]agentmodel.OtherConnectionSettings {
	if apiConns == nil {
		return nil
	}

	result := make(map[string]agentmodel.OtherConnectionSettings)

	for name, conn := range apiConns {
		result[name] = agentmodel.OtherConnectionSettings{
			DestinationEndpoint: conn.DestinationEndpoint,
			Headers:             conn.Headers,
			CertificateName:     conn.CertificateName,
		}
	}

	return result
}

func (mapper *Mapper) mapOpAMPConnectionToAPI(conn *agentmodel.OpAMPConnectionSettings) v1.OpAMPConnectionSettings {
	if conn == nil {
		return v1.OpAMPConnectionSettings{
			DestinationEndpoint: "",
			Headers:             nil,
			CertificateName:     nil,
		}
	}

	return v1.OpAMPConnectionSettings{
		DestinationEndpoint: conn.DestinationEndpoint,
		Headers:             conn.Headers,
		CertificateName:     conn.CertificateName,
	}
}

func (mapper *Mapper) mapTelemetryConnectionToAPI(
	conn *agentmodel.TelemetryConnectionSettings,
) v1.TelemetryConnectionSettings {
	if conn == nil {
		return v1.TelemetryConnectionSettings{
			DestinationEndpoint: "",
			Headers:             nil,
			CertificateName:     nil,
		}
	}

	return v1.TelemetryConnectionSettings{
		DestinationEndpoint: conn.DestinationEndpoint,
		Headers:             conn.Headers,
		CertificateName:     conn.CertificateName,
	}
}

func (mapper *Mapper) mapOtherConnectionsToAPI(
	conns map[string]agentmodel.OtherConnectionSettings,
) map[string]v1.OtherConnectionSettings {
	if conns == nil {
		return nil
	}

	result := make(map[string]v1.OtherConnectionSettings)

	for name, conn := range conns {
		result[name] = v1.OtherConnectionSettings{
			DestinationEndpoint: conn.DestinationEndpoint,
			Headers:             conn.Headers,
			CertificateName:     conn.CertificateName,
		}
	}

	return result
}

func (mapper *Mapper) mapAgentGroupConditionsToAPI(conditions []model.Condition) []v1.Condition {
	if len(conditions) == 0 {
		return nil
	}

	apiConditions := make([]v1.Condition, len(conditions))

	for i, condition := range conditions {
		apiConditions[i] = v1.Condition{
			Type:               v1.ConditionType(condition.Type),
			LastTransitionTime: v1.NewTime(condition.LastTransitionTime),
			Status:             v1.ConditionStatus(condition.Status),
			Reason:             condition.Reason,
			Message:            condition.Message,
		}
	}

	return apiConditions
}

func p[T any](v T) *T {
	return &v
}

func mapDeletedAtPtrToAPI(t *time.Time) *v1.Time {
	if t == nil {
		return nil
	}

	return p(v1.NewTime(*t))
}
