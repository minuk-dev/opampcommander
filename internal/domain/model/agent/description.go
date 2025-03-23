package agent

import "github.com/minuk-dev/opampcommander/internal/domain/model/vo"

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

func (ad *Description) Service() vo.Service {
	return vo.Service{
		Name:       ad.IdentifyingAttributes["service.name"],
		Namespace:  ad.IdentifyingAttributes["service.namespace"],
		Version:    ad.IdentifyingAttributes["service.version"],
		InstanceID: ad.IdentifyingAttributes["service.instance.id"],
	}
}

func (ad *Description) Host() Host {
	return Host{
		Name: ad.NonIdentifyingAttributes["host.name"],
	}
}
