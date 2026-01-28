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

	var agentRemoteConfig *model.AgentRemoteConfig

	if apiAgentGroup.Spec.AgentConfig != nil {
		agentRemoteConfig = &model.AgentRemoteConfig{
			Value:       []byte(apiAgentGroup.Spec.AgentConfig.Value),
			ContentType: apiAgentGroup.Spec.AgentConfig.ContentType,
		}
	}

	var agentConnectionConfig *model.AgentConnectionConfig

	if apiAgentGroup.Spec.AgentConfig != nil && apiAgentGroup.Spec.AgentConfig.ConnectionSettings != nil {
		connSettings := apiAgentGroup.Spec.AgentConfig.ConnectionSettings
		agentConnectionConfig = &model.AgentConnectionConfig{
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
			Priority:   apiAgentGroup.Metadata.Priority,
			Attributes: model.OfAttributes(apiAgentGroup.Metadata.Attributes),
			Selector: model.AgentSelector{
				IdentifyingAttributes:    apiAgentGroup.Metadata.Selector.IdentifyingAttributes,
				NonIdentifyingAttributes: apiAgentGroup.Metadata.Selector.NonIdentifyingAttributes,
			},
		},
		Spec: model.AgentGroupSpec{
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

	if domainAgentGroup.Spec.AgentRemoteConfig != nil {
		//exhaustruct:ignore
		agentConfig = &v1.AgentConfig{
			Value:       string(domainAgentGroup.Spec.AgentRemoteConfig.Value),
			ContentType: domainAgentGroup.Spec.AgentRemoteConfig.ContentType,
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
			Priority:   domainAgentGroup.Metadata.Priority,
			Attributes: v1.Attributes(domainAgentGroup.Metadata.Attributes),
			Selector: v1.AgentSelector{
				IdentifyingAttributes:    domainAgentGroup.Metadata.Selector.IdentifyingAttributes,
				NonIdentifyingAttributes: domainAgentGroup.Metadata.Selector.NonIdentifyingAttributes,
			},
		},
		Spec: v1.Spec{
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
			NewInstanceUID: mapper.mapNewInstanceUIDFromAPI(apiAgent.Spec.NewInstanceUID),
			RemoteConfig:   mapper.mapRemoteConfigFromAPI(&apiAgent.Spec.RemoteConfig),
			RestartInfo:    mapper.mapRestartInfoFromAPI(apiAgent.Spec.RestartRequiredAt),
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
			RemoteConfig:      mapper.mapRemoteConfigToAPI(&agent.Spec.RemoteConfig),
			RestartRequiredAt: mapper.mapRestartRequiredAtToAPI(agent.Spec.RestartInfo.RequiredRestartedAt),
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
	return &v1.AgentPackage{
		Metadata: v1.AgentPackageMetadata{
			Name:       agentPackage.Metadata.Name,
			Attributes: v1.Attributes(agentPackage.Metadata.Attributes),
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

func (mapper *Mapper) mapCustomCapabilitiesFromAPI(
	apiCustomCapabilities *v1.AgentCustomCapabilities,
) model.AgentCustomCapabilities {
	return model.AgentCustomCapabilities{
		Capabilities: apiCustomCapabilities.Capabilities,
	}
}

func (mapper *Mapper) mapRemoteConfigFromAPI(
	apiRemoteConfig *v1.AgentRemoteConfig,
) model.AgentSpecRemoteConfig {
	if apiRemoteConfig == nil || len(apiRemoteConfig.ConfigMap) == 0 {
		//nolint:exhaustruct // RemoteConfig will be nil for empty config
		return model.AgentSpecRemoteConfig{}
	}

	// Extract remote config names from the config map keys
	remoteConfigNames := make([]string, 0, len(apiRemoteConfig.ConfigMap))
	for name := range apiRemoteConfig.ConfigMap {
		remoteConfigNames = append(remoteConfigNames, name)
	}

	return model.AgentSpecRemoteConfig{
		RemoteConfig: remoteConfigNames,
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

func (mapper *Mapper) mapRestartInfoFromAPI(restartRequiredAt *v1.Time) model.AgentRestartInfo {
	if restartRequiredAt == nil || restartRequiredAt.IsZero() {
		//exhaustruct:ignore
		return model.AgentRestartInfo{}
	}

	return model.AgentRestartInfo{
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

func (mapper *Mapper) mapRemoteConfigToAPI(remoteConfig *model.AgentSpecRemoteConfig) v1.AgentRemoteConfig {
	configMap := make(map[string]v1.AgentConfigFile)

	// Add each remote config name to the config map
	for _, name := range remoteConfig.RemoteConfig {
		//nolint:exhaustruct // Body and ContentType are not needed for listing
		configMap[name] = v1.AgentConfigFile{}
	}

	return v1.AgentRemoteConfig{
		ConfigMap:  configMap,
		ConfigHash: "",
	}
}

func (mapper *Mapper) mapCustomCapabilitiesToAPI(
	customCapabilities *model.AgentCustomCapabilities,
) v1.AgentCustomCapabilities {
	return v1.AgentCustomCapabilities{
		Capabilities: customCapabilities.Capabilities,
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

func (mapper *Mapper) mapRestartRequiredAtToAPI(restartRequiredAt time.Time) *v1.Time {
	if restartRequiredAt.IsZero() {
		return nil
	}

	t := v1.NewTime(restartRequiredAt)

	return &t
}

func (mapper *Mapper) mapOpAMPConnectionFromAPI(apiConn v1.OpAMPConnectionSettings) model.OpAMPConnectionSettings {
	return model.OpAMPConnectionSettings{
		DestinationEndpoint: apiConn.DestinationEndpoint,
		Headers:             apiConn.Headers,
		Certificate:         mapper.mapTLSCertificateFromAPI(apiConn.Certificate),
	}
}

func (mapper *Mapper) mapTelemetryConnectionFromAPI(
	apiConn v1.TelemetryConnectionSettings,
) model.TelemetryConnectionSettings {
	return model.TelemetryConnectionSettings{
		DestinationEndpoint: apiConn.DestinationEndpoint,
		Headers:             apiConn.Headers,
		Certificate:         mapper.mapTLSCertificateFromAPI(apiConn.Certificate),
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
			Certificate:         mapper.mapTLSCertificateFromAPI(conn.Certificate),
		}
	}

	return result
}

func (mapper *Mapper) mapTLSCertificateFromAPI(apiCert v1.TLSCertificate) model.TelemetryTLSCertificate {
	return model.TelemetryTLSCertificate{
		Cert:       []byte(apiCert.Cert),
		PrivateKey: []byte(apiCert.PrivateKey),
		CaCert:     []byte(apiCert.CaCert),
	}
}

func (mapper *Mapper) mapOpAMPConnectionToAPI(conn model.OpAMPConnectionSettings) v1.OpAMPConnectionSettings {
	return v1.OpAMPConnectionSettings{
		DestinationEndpoint: conn.DestinationEndpoint,
		Headers:             conn.Headers,
		Certificate:         mapper.mapTLSCertificateToAPI(conn.Certificate),
	}
}

func (mapper *Mapper) mapTelemetryConnectionToAPI(
	conn model.TelemetryConnectionSettings,
) v1.TelemetryConnectionSettings {
	return v1.TelemetryConnectionSettings{
		DestinationEndpoint: conn.DestinationEndpoint,
		Headers:             conn.Headers,
		Certificate:         mapper.mapTLSCertificateToAPI(conn.Certificate),
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
			Certificate:         mapper.mapTLSCertificateToAPI(conn.Certificate),
		}
	}

	return result
}

func (mapper *Mapper) mapTLSCertificateToAPI(cert model.TelemetryTLSCertificate) v1.TLSCertificate {
	return v1.TLSCertificate{
		Cert:       string(cert.Cert),
		PrivateKey: string(cert.PrivateKey),
		CaCert:     string(cert.CaCert),
	}
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
