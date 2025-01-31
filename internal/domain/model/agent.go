package model

// Agent is a domain model to control opamp agent by opampcommander
type Agent struct {
	InstanceUUID    string
	Capabilities    *AgentCapabilities
	Description     *AgentDescription
	EffectiveConfig *AgentEffectiveConfig
	PacakgeStatuses *AgentPackageStatuses
	ComponentHealth *AgentComponentHealth
}

type AgentDescription struct {
	IdentifyingAttributes    map[string]string
	NonIdentifyingAttributes map[string]string
}

type (
	AgentCapabilities    struct{}
	AgentEffectiveConfig struct{}
	AgentPackageStatuses struct{}
	AgentComponentHealth struct{}
)

type OS struct {
	Type    string
	Version string
}

type Service struct {
	Name       string
	Namespace  string
	Version    string
	InstanceID string
}

type AgentHost struct {
	Name string
}

// OS is a required field of AgentDescription
// ref. https://github.com/open-telemetry/opamp-spec/blob/main/specification.md#agentdescriptionnon_identifying_attributes
func (ad *AgentDescription) OS() OS {
	return OS{
		Type:    ad.NonIdentifyingAttributes["os.type"],
		Version: ad.NonIdentifyingAttributes["os.version"],
	}
}

func (ad *AgentDescription) Service() Service {
	return Service{
		Name:       ad.IdentifyingAttributes["service.name"],
		Namespace:  ad.IdentifyingAttributes["service.namespace"],
		Version:    ad.IdentifyingAttributes["service.version"],
		InstanceID: ad.IdentifyingAttributes["service.instance.id"],
	}
}

func (ad *AgentDescription) Host() AgentHost {
	return AgentHost{
		Name: ad.NonIdentifyingAttributes["host.name"],
	}
}
