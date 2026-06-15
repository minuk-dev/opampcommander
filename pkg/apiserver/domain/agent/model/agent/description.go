package agent

import "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model/vo"

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

// Host returns host information derived from the "host.*" resource attributes.
func (ad *Description) Host() Host {
	return Host{
		ID:   ad.attr("host.id"),
		Name: ad.attr("host.name"),
		Arch: ad.attr("host.arch"),
		Type: ad.attr("host.type"),
	}
}

// Container returns container information derived from the "container.*"
// resource attributes.
func (ad *Description) Container() Container {
	return Container{
		ID:        ad.attr("container.id"),
		Name:      ad.attr("container.name"),
		ImageName: ad.attr("container.image.name"),
		Runtime:   ad.attr("container.runtime"),
	}
}

// K8s returns Kubernetes information derived from the "k8s.*" resource
// attributes.
func (ad *Description) K8s() K8s {
	return K8s{
		PodName:       ad.attr("k8s.pod.name"),
		PodUID:        ad.attr("k8s.pod.uid"),
		NamespaceName: ad.attr("k8s.namespace.name"),
		NodeName:      ad.attr("k8s.node.name"),
		ContainerName: ad.attr("k8s.container.name"),
	}
}

// Cloud returns cloud information derived from the "cloud.*" resource
// attributes.
func (ad *Description) Cloud() Cloud {
	return Cloud{
		Provider: ad.attr("cloud.provider"),
		Platform: ad.attr("cloud.platform"),
		Region:   ad.attr("cloud.region"),
	}
}

// EnvironmentKind classifies the compute substrate the agent runs on. Any
// container/k8s signal makes it a container; otherwise any host signal makes it
// a host; otherwise it is unknown.
func (ad *Description) EnvironmentKind() EnvironmentKind {
	if !ad.Container().IsZero() || !ad.K8s().IsZero() {
		return EnvironmentKindContainer
	}

	if !ad.Host().IsZero() {
		return EnvironmentKindHost
	}

	return EnvironmentKindUnknown
}

// Platform classifies the orchestration/deployment environment. Kubernetes is
// detected from "k8s.*" attributes; ECS from "cloud.platform"; a standalone
// container runtime from "container.*"; a VM from a host with a cloud provider;
// and a bare-metal machine from a host with no cloud provider.
func (ad *Description) Platform() Platform {
	if !ad.K8s().IsZero() {
		return PlatformKubernetes
	}

	if ad.Cloud().Platform == "aws_ecs" {
		return PlatformECS
	}

	if !ad.Container().IsZero() {
		return PlatformDocker
	}

	if !ad.Host().IsZero() {
		if ad.Cloud().Provider != "" {
			return PlatformVM
		}

		return PlatformBareMetal
	}

	return PlatformUnknown
}

// attr returns the value for key, preferring non-identifying attributes and
// falling back to identifying attributes. The host/container/k8s/cloud
// descriptors are conventionally non-identifying, but agents and OpAMP SDKs vary
// in which bucket they place a given resource attribute, so discovery tolerates
// either to avoid silently missing an environment.
func (ad *Description) attr(key string) string {
	if value, ok := ad.NonIdentifyingAttributes[key]; ok && value != "" {
		return value
	}

	return ad.IdentifyingAttributes[key]
}
