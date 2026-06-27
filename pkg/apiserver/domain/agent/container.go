package agentmodel

import (
	"time"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

// Container is a domain aggregate that represents the container one or more
// agents run in. Containers are not created by users; they are discovered and
// upserted from the OpenTelemetry "container.*"/"k8s.*" attributes an agent
// reports in its description.
type Container struct {
	Metadata ContainerMetadata
	Spec     ContainerSpec
	Status   ContainerStatus
}

// ContainerMetadata holds identity and lifecycle information for a container.
type ContainerMetadata struct {
	// ID is the stable identity of the container: the OpenTelemetry
	// "k8s.pod.uid" attribute, falling back to "container.id" (which is not
	// stable across container restarts).
	ID string
	// Name is the reported "container.name", falling back to "k8s.pod.name".
	Name string
	// Labels and Annotations are reserved for user-supplied metadata.
	Labels      map[string]string
	Annotations map[string]string
	// FirstSeenAt is when the container was first discovered.
	FirstSeenAt time.Time
	// LastSeenAt is the most recent time an agent in this container reported.
	LastSeenAt time.Time
}

// ContainerSpec holds the discovered, descriptive facts about a container.
type ContainerSpec struct {
	// Platform classifies the deployment environment (docker, kubernetes, ...).
	Platform agent.Platform
	// ImageName is the reported "container.image.name".
	ImageName string
	// Runtime is the reported "container.runtime".
	Runtime string
	// HostID links this container to the Host (node) it runs on, when known.
	// It is the host identity reported by the same agent, falling back to the
	// Kubernetes node name.
	HostID string
	// K8s is the reported Kubernetes context, if any.
	K8s agent.K8s
}

// ContainerStatus holds the observed state of a container.
type ContainerStatus struct {
	// AgentInstanceUIDs are the agents currently associated with this container.
	AgentInstanceUIDs []uuid.UUID
	// Conditions is a list of conditions that apply to the container.
	Conditions []model.Condition
}

// ContainerIDOf returns the stable identity for the container described by desc:
// the "k8s.pod.uid" attribute, falling back to "container.id". It returns an
// empty string when the description carries no container or pod attributes.
func ContainerIDOf(desc agent.Description) string {
	if uid := desc.K8s().PodUID; uid != "" {
		return uid
	}

	return desc.Container().ID
}

// NewContainer creates a new, empty container with the given identity. Use
// ObserveAgent to populate it from an agent's reported description.
func NewContainer(id string, now time.Time) *Container {
	return &Container{
		Metadata: ContainerMetadata{
			ID:          id,
			Name:        "",
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
			FirstSeenAt: now,
			LastSeenAt:  now,
		},
		Spec: ContainerSpec{
			Platform:  agent.PlatformUnknown,
			ImageName: "",
			Runtime:   "",
			HostID:    "",
			K8s: agent.K8s{
				PodName:       "",
				PodUID:        "",
				NamespaceName: "",
				NodeName:      "",
				ContainerName: "",
			},
		},
		Status: ContainerStatus{
			AgentInstanceUIDs: nil,
			Conditions:        nil,
		},
	}
}

// ObserveAgent refreshes the container's discovered spec from an agent's
// description, advances LastSeenAt, and ensures the agent is associated with
// this container.
func (c *Container) ObserveAgent(instanceUID uuid.UUID, desc agent.Description, now time.Time) {
	container := desc.Container()
	k8s := desc.K8s()

	if name := containerName(container, k8s); name != "" {
		c.Metadata.Name = name
	}

	c.Spec.Platform = desc.Platform()
	c.Spec.ImageName = container.ImageName
	c.Spec.Runtime = container.Runtime
	c.Spec.HostID = containerHostID(desc)
	c.Spec.K8s = k8s

	c.Metadata.LastSeenAt = now
	c.Status.AgentInstanceUIDs = appendUniqueUID(c.Status.AgentInstanceUIDs, instanceUID)
}

// containerName picks the human-facing name for a container, preferring the
// container name and falling back to the pod name.
func containerName(container agent.Container, k8s agent.K8s) string {
	if container.Name != "" {
		return container.Name
	}

	return k8s.PodName
}

// containerHostID resolves the host a container runs on: the host identity
// reported by the same agent, falling back to the Kubernetes node name.
func containerHostID(desc agent.Description) string {
	if id := HostIDOf(desc); id != "" {
		return id
	}

	return desc.K8s().NodeName
}
