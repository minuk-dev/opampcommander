// Package mapper provides functions to map between api and domain models.
package mapper

import (
	"time"

	"github.com/samber/lo"

	v1agent "github.com/minuk-dev/opampcommander/api/v1/agent"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

// Mapper is a struct that provides methods to map between domain models and API models.
type Mapper struct{}

// New creates a new instance of Mapper.
func New() *Mapper {
	return &Mapper{}
}

// MapAgentToAPI maps a domain model Agent to an API model Agent.
func (mapper *Mapper) MapAgentToAPI(agent *model.Agent) *v1agent.Agent {
	return &v1agent.Agent{
		Metadata: v1agent.Metadata{
			InstanceUID: agent.Metadata.InstanceUID,
			Description: v1agent.Description{
				IdentifyingAttributes:    agent.Metadata.Description.IdentifyingAttributes,
				NonIdentifyingAttributes: agent.Metadata.Description.NonIdentifyingAttributes,
			},
			Capabilities:       v1agent.Capabilities(agent.Metadata.Capabilities),
			CustomCapabilities: mapper.mapCustomCapabilitiesToAPI(&agent.Metadata.CustomCapabilities),
		},
		Spec: v1agent.Spec{
			RemoteConfig: mapper.mapRemoteConfigToAPI(&agent.Spec.RemoteConfig),
		},
		Status: v1agent.Status{
			EffectiveConfig: v1agent.EffectiveConfig{
				ConfigMap: v1agent.ConfigMap{
					ConfigMap: lo.MapValues(agent.Status.EffectiveConfig.ConfigMap.ConfigMap,
						func(value model.AgentConfigFile, _ string) v1agent.ConfigFile {
							return mapper.mapConfigFileToAPI(value)
						}),
				},
			},
			PackageStatuses: v1agent.PackageStatuses{
				Packages: lo.MapValues(agent.Status.PackageStatuses.Packages,
					func(value model.AgentPackageStatus, _ string) v1agent.PackageStatus {
						return v1agent.PackageStatus{
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

func (mapper *Mapper) mapComponentHealthToAPI(health *model.AgentComponentHealth) v1agent.ComponentHealth {
	componentsMap := make(map[string]string)
	for name, comp := range health.ComponentHealthMap {
		componentsMap[name] = comp.Status
	}

	return v1agent.ComponentHealth{
		Healthy:       health.Healthy,
		StartTimeUnix: health.StartTime.Unix(),
		LastError:     health.LastError,
		Status:        health.Status,
		StatusTimeMS:  health.StatusTime.UnixMilli(),
		ComponentsMap: componentsMap,
	}
}

func (mapper *Mapper) mapRemoteConfigToAPI(remoteConfig *model.RemoteConfig) v1agent.RemoteConfig {
	configMap := make(map[string]v1agent.ConfigFile)

	// If there's config data, add it to the config map
	if len(remoteConfig.ConfigData.Config) > 0 {
		configMap["remote_config.yaml"] = v1agent.ConfigFile{
			Body:        string(remoteConfig.ConfigData.Config),
			ContentType: TextYAML,
		}
	}

	return v1agent.RemoteConfig{
		ConfigMap:  configMap,
		ConfigHash: string(remoteConfig.ConfigData.Key),
	}
}

func (mapper *Mapper) mapCustomCapabilitiesToAPI(
	customCapabilities *model.AgentCustomCapabilities,
) v1agent.CustomCapabilities {
	return v1agent.CustomCapabilities{
		Capabilities: customCapabilities.Capabilities,
	}
}

func (mapper *Mapper) mapAvailableComponentsToAPI(
	availableComponents *model.AgentAvailableComponents,
) v1agent.AvailableComponents {
	components := lo.MapValues(availableComponents.Components,
		func(value model.ComponentDetails, _ string) v1agent.ComponentDetails {
			// Extract type and version from metadata if available
			componentType := value.Metadata["type"]
			version := value.Metadata["version"]

			return v1agent.ComponentDetails{
				Type:    componentType,
				Version: version,
			}
		})

	return v1agent.AvailableComponents{
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

func (mapper *Mapper) mapConfigFileToAPI(configFile model.AgentConfigFile) v1agent.ConfigFile {
	switch configFile.ContentType {
	case TextJSON,
		TextYAML,
		Empty:
		return v1agent.ConfigFile{
			Body:        string(configFile.Body),
			ContentType: configFile.ContentType,
		}
	default:
		return v1agent.ConfigFile{
			Body:        "unsupported content type",
			ContentType: configFile.ContentType,
		}
	}
}

func (mapper *Mapper) mapConditionsToAPI(conditions []model.AgentCondition) []v1agent.Condition {
	if len(conditions) == 0 {
		return nil
	}

	apiConditions := make([]v1agent.Condition, len(conditions))
	for i, condition := range conditions {
		apiConditions[i] = v1agent.Condition{
			Type:               v1agent.ConditionType(condition.Type),
			LastTransitionTime: mapper.formatTime(condition.LastTransitionTime),
			Status:             v1agent.ConditionStatus(condition.Status),
			Reason:             condition.Reason,
			Message:            condition.Message,
		}
	}

	return apiConditions
}
