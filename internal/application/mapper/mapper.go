// Package mapper provides functions to map between api and domain models.
package mapper

import (
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
		InstanceUID:  agent.Metadata.InstanceUID,
		IsManaged:    agent.Spec.RemoteConfig.IsManaged(),
		Capabilities: v1agent.Capabilities(agent.Metadata.Capabilities),
		Description: v1agent.Description{
			IdentifyingAttributes:    agent.Metadata.Description.IdentifyingAttributes,
			NonIdentifyingAttributes: agent.Metadata.Description.NonIdentifyingAttributes,
		},
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
		ComponentHealth:     v1agent.ComponentHealth{},
		RemoteConfig:        v1agent.RemoteConfig{},
		CustomCapabilities:  v1agent.CustomCapabilities{},
		AvailableComponents: v1agent.AvailableComponents{},
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
