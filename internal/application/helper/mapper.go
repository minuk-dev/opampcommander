// Package helper provides helper functions for mapping between domain models and API models.
package helper

import (
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/agent"
)

// Mapper is a struct that provides methods to map between domain models and API models.
type Mapper struct{}

// NewMapper creates a new instance of Mapper.
func NewMapper() *Mapper {
	return &Mapper{}
}

// MapAPIToAgentGroup maps an API model AgentGroup to a domain model AgentGroup.
func (mapper *Mapper) MapAPIToAgentGroup(apiAgentGroup *v1.AgentGroup) *model.AgentGroup {
	if apiAgentGroup == nil {
		return nil
	}

	var agentRemoteConfig *model.AgentGroupAgentRemoteConfig

	if apiAgentGroup.Spec.AgentConfig != nil && apiAgentGroup.Spec.AgentConfig.AgentRemoteConfig != nil {
		apiRemoteConfig := apiAgentGroup.Spec.AgentConfig.AgentRemoteConfig

		agentRemoteConfig = &model.AgentGroupAgentRemoteConfig{
			AgentRemoteConfigName: apiRemoteConfig.AgentRemoteConfigName,
			AgentRemoteConfigRef:  apiRemoteConfig.AgentRemoteConfigRef,
			AgentRemoteConfigSpec: nil,
		}
		if apiRemoteConfig.AgentRemoteConfigSpec != nil {
			agentRemoteConfig.AgentRemoteConfigSpec = &model.AgentRemoteConfigSpec{
				Value:       []byte(apiRemoteConfig.AgentRemoteConfigSpec.Value),
				ContentType: apiRemoteConfig.AgentRemoteConfigSpec.ContentType,
			}
		}
	}

	var agentConnectionConfig *model.AgentGroupConnectionConfig

	if apiAgentGroup.Spec.AgentConfig != nil && apiAgentGroup.Spec.AgentConfig.ConnectionSettings != nil {
		connSettings := apiAgentGroup.Spec.AgentConfig.ConnectionSettings
		agentConnectionConfig = &model.AgentGroupConnectionConfig{
			OpAMPConnection:  mapper.mapOpAMPConnectionFromAPI(connSettings.OpAMP),
			OwnMetrics:       mapper.mapTelemetryConnectionFromAPI(connSettings.OwnMetrics),
			OwnLogs:          mapper.mapTelemetryConnectionFromAPI(connSettings.OwnLogs),
			OwnTraces:        mapper.mapTelemetryConnectionFromAPI(connSettings.OwnTraces),
			OtherConnections: mapper.mapOtherConnectionsFromAPI(connSettings.OtherConnections),
		}
	}

	//exhaustruct:ignore
	return &model.AgentGroup{
		Metadata: model.AgentGroupMetadata{
			Name:       apiAgentGroup.Metadata.Name,
			Attributes: model.OfAttributes(apiAgentGroup.Metadata.Attributes),
		},
		Spec: model.AgentGroupSpec{
			Priority: apiAgentGroup.Spec.Priority,
			Selector: model.AgentSelector{
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
func (mapper *Mapper) MapAgentGroupToAPI(domainAgentGroup *model.AgentGroup) *v1.AgentGroup {
	if domainAgentGroup == nil {
		return nil
	}

	var agentConfig *v1.AgentConfig

	if domainAgentGroup.Spec.AgentRemoteConfig != nil || domainAgentGroup.Spec.AgentConnectionConfig != nil {
		//exhaustruct:ignore
		agentConfig = &v1.AgentConfig{}

		if domainAgentGroup.Spec.AgentRemoteConfig != nil {
			agentConfig.AgentRemoteConfig = &v1.AgentGroupRemoteConfig{
				AgentRemoteConfigName: domainAgentGroup.Spec.AgentRemoteConfig.AgentRemoteConfigName,
				AgentRemoteConfigRef:  domainAgentGroup.Spec.AgentRemoteConfig.AgentRemoteConfigRef,
				AgentRemoteConfigSpec: nil,
			}
			if domainAgentGroup.Spec.AgentRemoteConfig.AgentRemoteConfigSpec != nil {
				agentConfig.AgentRemoteConfig.AgentRemoteConfigSpec = &v1.AgentRemoteConfigSpec{
					Value:       string(domainAgentGroup.Spec.AgentRemoteConfig.AgentRemoteConfigSpec.Value),
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
			Name:       domainAgentGroup.Metadata.Name,
			CreatedAt:  p(v1.NewTime(domainAgentGroup.Metadata.CreatedAt)),
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
func (mapper *Mapper) MapAPIToAgent(apiAgent *v1.Agent) *model.Agent {
	//exhaustruct:ignore
	return &model.Agent{
		Metadata: model.AgentMetadata{
			InstanceUID: apiAgent.Metadata.InstanceUID,
			Description: agent.Description{
				IdentifyingAttributes:    apiAgent.Metadata.Description.IdentifyingAttributes,
				NonIdentifyingAttributes: apiAgent.Metadata.Description.NonIdentifyingAttributes,
			},
			Capabilities:       agent.Capabilities(apiAgent.Metadata.Capabilities),
			CustomCapabilities: mapper.mapCustomCapabilitiesFromAPI(&apiAgent.Metadata.CustomCapabilities),
		},
		//exhaustruct:ignore
		Spec: model.AgentSpec{
			NewInstanceUID:    mapper.mapNewInstanceUIDFromAPI(apiAgent.Spec.NewInstanceUID),
			RemoteConfig:      mapper.mapRemoteConfigFromAPI(&apiAgent.Spec.RemoteConfig),
			PackagesAvailable: mapper.mapPackagesAvailableFromAPI(&apiAgent.Spec.PackagesAvailable),
			RestartInfo:       mapper.mapRestartInfoFromAPI(apiAgent.Spec.RestartRequiredAt),
		},
		// Note: Status is not mapped here as it is usually managed by the system.
	}
}

// MapAgentToAPI maps a domain model Agent to an API model Agent.
func (mapper *Mapper) MapAgentToAPI(agent *model.Agent) *v1.Agent {
	return &v1.Agent{
		Metadata: v1.AgentMetadata{
			InstanceUID: agent.Metadata.InstanceUID,
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
						func(value model.AgentConfigFile, _ string) v1.AgentConfigFile {
							return mapper.mapConfigFileToAPI(value)
						}),
				},
			},
			PackageStatuses: v1.AgentPackageStatuses{
				Packages: lo.MapValues(agent.Status.PackageStatuses.Packages,
					func(value model.AgentPackageStatusEntry, _ string) v1.AgentStatusPackageEntry {
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
func (mapper *Mapper) MapAgentPackageToAPI(agentPackage *model.AgentPackage) *v1.AgentPackage {
	var deletedAt *v1.Time
	if agentPackage.Metadata.DeletedAt != nil {
		t := v1.NewTime(*agentPackage.Metadata.DeletedAt)
		deletedAt = &t
	}

	var createdAt *v1.Time
	if agentPackage.Metadata.CreatedAt != nil {
		t := v1.NewTime(*agentPackage.Metadata.CreatedAt)
		createdAt = &t
	}

	return &v1.AgentPackage{
		Metadata: v1.AgentPackageMetadata{
			Name:       agentPackage.Metadata.Name,
			Attributes: v1.Attributes(agentPackage.Metadata.Attributes),
			CreatedAt:  createdAt,
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
func (mapper *Mapper) MapAPIToAgentPackage(apiModel *v1.AgentPackage) *model.AgentPackage {
	return &model.AgentPackage{
		Metadata: model.AgentPackageMetadata{
			Name:       apiModel.Metadata.Name,
			Attributes: model.OfAttributes(apiModel.Metadata.Attributes),
			CreatedAt:  &apiModel.Metadata.CreatedAt.Time,
			DeletedAt:  nil,
		},
		Spec: model.AgentPackageSpec{
			PackageType: apiModel.Spec.PackageType,
			Version:     apiModel.Spec.Version,
			DownloadURL: apiModel.Spec.DownloadURL,
			ContentHash: apiModel.Spec.ContentHash,
			Signature:   apiModel.Spec.Signature,
			Headers:     apiModel.Spec.Headers,
			Hash:        apiModel.Spec.Hash,
		},
		Status: model.AgentPackageStatus{
			// Skip mapping conditions as they are usually managed by the system.
			Conditions: nil,
		},
	}
}

// MapCertificateToAPI maps a domain model Certificate to an API model Certificate.
func (mapper *Mapper) MapCertificateToAPI(domain *model.Certificate) *v1.Certificate {
	if domain == nil {
		return nil
	}

	return &v1.Certificate{
		Kind:       v1.CertificateKind,
		APIVersion: v1.APIVersion,
		Metadata: v1.CertificateMetadata{
			Name:       domain.Metadata.Name,
			Attributes: v1.Attributes(domain.Metadata.Attributes),
			CreatedAt:  p(v1.NewTime(domain.Metadata.CreatedAt)),
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
func (mapper *Mapper) MapAPIToCertificate(api *v1.Certificate) *model.Certificate {
	if api == nil {
		return nil
	}

	var createdAt time.Time
	if api.Metadata.CreatedAt != nil {
		createdAt = api.Metadata.CreatedAt.Time
	}

	var deletedAt time.Time
	if api.Metadata.DeletedAt != nil {
		deletedAt = api.Metadata.DeletedAt.Time
	}

	return &model.Certificate{
		Metadata: model.CertificateMetadata{
			Name:       api.Metadata.Name,
			Attributes: model.Attributes(api.Metadata.Attributes),
			CreatedAt:  createdAt,
			DeletedAt:  deletedAt,
		},
		Spec: model.CertificateSpec{
			Cert:       []byte(api.Spec.Cert),
			PrivateKey: []byte(api.Spec.PrivateKey),
			CaCert:     []byte(api.Spec.CaCert),
		},
		Status: model.CertificateStatus{
			Conditions: nil,
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

func (mapper *Mapper) mapCustomCapabilitiesFromAPI(
	apiCustomCapabilities *v1.AgentCustomCapabilities,
) model.AgentCustomCapabilities {
	return model.AgentCustomCapabilities{
		Capabilities: apiCustomCapabilities.Capabilities,
	}
}

func (mapper *Mapper) mapRemoteConfigFromAPI(
	apiRemoteConfig *v1.AgentSpecRemoteConfig,
) *model.AgentSpecRemoteConfig {
	if apiRemoteConfig == nil || len(apiRemoteConfig.RemoteConfigNames) == 0 {
		return nil
	}

	// Convert RemoteConfigNames to ConfigMap with empty placeholders
	// The actual config values will be fetched when needed
	configMap := make(map[string]model.AgentConfigFile)
	for _, name := range apiRemoteConfig.RemoteConfigNames {
		configMap[name] = model.AgentConfigFile{
			Body:        nil,
			ContentType: "",
		}
	}

	return &model.AgentSpecRemoteConfig{
		ConfigMap: model.AgentConfigMap{
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

func (mapper *Mapper) mapRestartInfoFromAPI(restartRequiredAt *v1.Time) *model.AgentRestartInfo {
	if restartRequiredAt == nil || restartRequiredAt.IsZero() {
		return nil
	}

	return &model.AgentRestartInfo{
		RequiredRestartedAt: restartRequiredAt.Time,
	}
}

func (mapper *Mapper) formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.Format(time.RFC3339)
}

func (mapper *Mapper) mapComponentHealthToAPI(health *model.AgentComponentHealth) v1.AgentComponentHealth {
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

func (mapper *Mapper) mapRemoteConfigToAPI(remoteConfig *model.AgentSpecRemoteConfig) v1.AgentSpecRemoteConfig {
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
	packagesAvailable *model.AgentSpecPackage,
) v1.AgentSpecPackages {
	if packagesAvailable == nil {
		//exhaustruct:ignore
		return v1.AgentSpecPackages{}
	}

	return v1.AgentSpecPackages{
		Packages: packagesAvailable.Packages,
	}
}

func (mapper *Mapper) mapRestartRequiredAtToAPI(restartInfo *model.AgentRestartInfo) *v1.Time {
	if restartInfo == nil || restartInfo.RequiredRestartedAt.IsZero() {
		return nil
	}

	t := v1.NewTime(restartInfo.RequiredRestartedAt)

	return &t
}

func (mapper *Mapper) mapCustomCapabilitiesToAPI(
	customCapabilities *model.AgentCustomCapabilities,
) v1.AgentCustomCapabilities {
	return v1.AgentCustomCapabilities{
		Capabilities: customCapabilities.Capabilities,
	}
}

func (mapper *Mapper) mapPackagesAvailableFromAPI(
	apiPackages *v1.AgentSpecPackages,
) *model.AgentSpecPackage {
	if apiPackages == nil {
		return nil
	}

	return &model.AgentSpecPackage{
		Packages: apiPackages.Packages,
	}
}

func (mapper *Mapper) mapAvailableComponentsToAPI(
	availableComponents *model.AgentAvailableComponents,
) v1.AgentAvailableComponents {
	components := lo.MapValues(availableComponents.Components,
		func(value model.ComponentDetails, _ string) v1.AgentComponentDetails {
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

func (mapper *Mapper) mapConfigFileToAPI(configFile model.AgentConfigFile) v1.AgentConfigFile {
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

func (mapper *Mapper) mapAgentConditionsToAPI(conditions []model.AgentCondition) []v1.Condition {
	return lo.Map(conditions, func(condition model.AgentCondition, _ int) v1.Condition {
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

func (mapper *Mapper) mapOpAMPConnectionFromAPI(apiConn v1.OpAMPConnectionSettings) *model.OpAMPConnectionSettings {
	return &model.OpAMPConnectionSettings{
		DestinationEndpoint: apiConn.DestinationEndpoint,
		Headers:             apiConn.Headers,
		CertificateName:     apiConn.CertificateName,
	}
}

func (mapper *Mapper) mapTelemetryConnectionFromAPI(
	apiConn v1.TelemetryConnectionSettings,
) *model.TelemetryConnectionSettings {
	return &model.TelemetryConnectionSettings{
		DestinationEndpoint: apiConn.DestinationEndpoint,
		Headers:             apiConn.Headers,
		CertificateName:     apiConn.CertificateName,
	}
}

func (mapper *Mapper) mapOtherConnectionsFromAPI(
	apiConns map[string]v1.OtherConnectionSettings,
) map[string]model.OtherConnectionSettings {
	if apiConns == nil {
		return nil
	}

	result := make(map[string]model.OtherConnectionSettings)

	for name, conn := range apiConns {
		result[name] = model.OtherConnectionSettings{
			DestinationEndpoint: conn.DestinationEndpoint,
			Headers:             conn.Headers,
			CertificateName:     conn.CertificateName,
		}
	}

	return result
}

func (mapper *Mapper) mapOpAMPConnectionToAPI(conn *model.OpAMPConnectionSettings) v1.OpAMPConnectionSettings {
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
	conn *model.TelemetryConnectionSettings,
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
	conns map[string]model.OtherConnectionSettings,
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
