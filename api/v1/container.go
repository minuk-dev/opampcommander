package v1

const (
	// ContainerKind is the kind of the container resource.
	ContainerKind = "Container"
)

// Container represents a container one or more agents run in. Containers are
// discovered from the OpenTelemetry attributes agents report, not created by
// users.
type Container struct {
	Kind       string            `json:"kind"`
	APIVersion string            `json:"apiVersion"`
	Metadata   ContainerMetadata `json:"metadata"`
	Spec       ContainerSpec     `json:"spec"`
	Status     ContainerStatus   `json:"status"`
} // @name Container

// ContainerMetadata contains identity and lifecycle information for a container.
type ContainerMetadata struct {
	// ID is the stable identity of the container (OpenTelemetry "k8s.pod.uid",
	// falling back to "container.id").
	ID          string            `json:"id"`
	Name        string            `json:"name,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	FirstSeenAt Time              `json:"firstSeenAt"`
	LastSeenAt  Time              `json:"lastSeenAt"`
} // @name ContainerMetadata

// ContainerSpec contains the discovered facts about a container.
type ContainerSpec struct {
	// Platform classifies the deployment environment (docker, kubernetes, ...).
	Platform  string `json:"platform"`
	ImageName string `json:"imageName,omitempty"`
	Runtime   string `json:"runtime,omitempty"`
	// HostID links this container to the host (node) it runs on, when known.
	HostID string `json:"hostId,omitempty"`
	// Kubernetes context, populated for kubernetes platforms.
	K8sPodName       string `json:"k8sPodName,omitempty"`
	K8sNamespaceName string `json:"k8sNamespaceName,omitempty"`
	K8sNodeName      string `json:"k8sNodeName,omitempty"`
} // @name ContainerSpec

// ContainerStatus contains the observed state of a container.
type ContainerStatus struct {
	// AgentInstanceUIDs are the agents currently associated with this container.
	AgentInstanceUIDs []string    `json:"agentInstanceUids"`
	Conditions        []Condition `json:"conditions,omitempty"`
} // @name ContainerStatus
