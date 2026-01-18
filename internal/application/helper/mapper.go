// Package helper provides helper functions for mapping between domain models and API models.
package helper

import (
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	v1agentgroup "github.com/minuk-dev/opampcommander/api/v1/agentgroup"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/agent"
)

// Mapper is a struct that provides methods to map between domain models and API models.
type Mapper struct{}

func (mapper *Mapper) MapAPIToAgentGroup(agentGroup *model.AgentGroup) v1agentgroup.AgentGroup {
	panic("unimplemented")
}

// NewMapper creates a new instance of Mapper.
func NewMapper() *Mapper {
	return &Mapper{}
}

// MapAPIToAgent maps an API model Agent to a domain model Agent.
func (mapper *Mapper) MapAPIToAgent(apiAgent *v1.Agent) *model.Agent {
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
		Spec: model.AgentSpec{
			NewInstanceUID: mapper.mapNewInstanceUIDFromAPI(apiAgent.Spec.NewInstanceUID),
			RemoteConfig:   mapper.mapRemoteConfigFromAPI(&apiAgent.Spec.RemoteConfig),
		},
		// Note: Status is not mapped here as it is usually managed by the system.
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
) model.RemoteConfig {
	configData, ok := apiRemoteConfig.ConfigMap["remote_config.yaml"]
	if !ok {
		return model.RemoteConfig{}
	}

	return model.RemoteConfig{
		Config: []byte(configData.Body),
		Hash:   []byte(apiRemoteConfig.ConfigHash),
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
			NewInstanceUID: mapper.mapNewInstanceUIDToAPI(agent.Spec.NewInstanceUID[:]),
			RemoteConfig:   mapper.mapRemoteConfigToAPI(&agent.Spec.RemoteConfig),
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
					func(value model.AgentPackageStatus, _ string) v1.AgentPackageStatus {
						return v1.AgentPackageStatus{
							Name: value.Name,
						}
					}),
				ServerProvidedAllPackagesHash: string(agent.Status.PackageStatuses.ServerProvidedAllPackgesHash),
				ErrorMessage:                  agent.Status.PackageStatuses.ErrorMessage,
			},
			ComponentHealth:     mapper.mapComponentHealthToAPI(&agent.Status.ComponentHealth),
			AvailableComponents: mapper.mapAvailableComponentsToAPI(&agent.Status.AvailableComponents),
			Conditions:          mapper.mapConditionsToAPI(agent.Status.Conditions),
			Connected:           agent.Status.Connected,
			ConnectionType:      agent.Status.ConnectionType.String(),
			SequenceNum:         agent.Status.SequenceNum,
			LastReportedAt:      mapper.formatTime(agent.Status.LastReportedAt),
		},
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

func (mapper *Mapper) mapRemoteConfigToAPI(remoteConfig *model.RemoteConfig) v1.AgentRemoteConfig {
	configMap := make(map[string]v1.AgentConfigFile)

	// If there's config data, add it to the config map
	if len(remoteConfig.Config) > 0 {
		configMap["remote_config.yaml"] = v1.AgentConfigFile{
			Body:        string(remoteConfig.Config),
			ContentType: TextYAML,
		}
	}

	return v1.AgentRemoteConfig{
		ConfigMap:  configMap,
		ConfigHash: string(remoteConfig.Hash),
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

func (mapper *Mapper) mapConditionsToAPI(conditions []model.AgentCondition) []v1.Condition {
	if len(conditions) == 0 {
		return nil
	}

	apiConditions := make([]v1.Condition, len(conditions))
	for i, condition := range conditions {
		apiConditions[i] = v1.Condition{
			Type:               v1.ConditionType(condition.Type),
			LastTransitionTime: mapper.formatTime(condition.LastTransitionTime),
			Status:             v1.ConditionStatus(condition.Status),
			Reason:             condition.Reason,
			Message:            condition.Message,
		}
	}

	return apiConditions
}

func (mapper *Mapper) mapNewInstanceUIDToAPI(newInstanceUID []byte) string {
	if len(newInstanceUID) == 0 {
		return ""
	}

	return string(newInstanceUID)
}
