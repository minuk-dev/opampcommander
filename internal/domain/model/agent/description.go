package agent

import "github.com/minuk-dev/opampcommander/internal/domain/model/vo"

// Description represents the description of an agent.
// It contains identifying and non-identifying attributes.
type Description struct {
	IdentifyingAttributes    map[string]string
	NonIdentifyingAttributes map[string]string
}

// OS is a required field of AgentDescription
// https://github.com/open-telemetry/opamp-spec/blob/main/specification.md#agentdescriptionnon_identifying_attributes
func (ad *Description) OS() vo.OS {
	return vo.OS{
		Type:    ad.NonIdentifyingAttributes["os.type"],
		Version: ad.NonIdentifyingAttributes["os.version"],
	}
}

// Service returns service information.
func (ad *Description) Service() vo.Service {
	return vo.Service{
		Name:       ad.IdentifyingAttributes["service.name"],
		Namespace:  ad.IdentifyingAttributes["service.namespace"],
		Version:    ad.IdentifyingAttributes["service.version"],
		InstanceID: ad.IdentifyingAttributes["service.instance.id"],
	}
}

// Host returns host information.
func (ad *Description) Host() Host {
	return Host{
		Name: ad.NonIdentifyingAttributes["host.name"],
	}
}
