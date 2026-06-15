package agent

// Container is a descriptor value object that represents the container an agent
// runs in. Its fields are derived from the OpenTelemetry "container.*"
// non-identifying resource attributes reported by the agent.
type Container struct {
	// ID is the OpenTelemetry "container.id" attribute. Note it is not stable
	// across container restarts, so the Container aggregate prefers "k8s.pod.uid"
	// for identity when available.
	ID string
	// Name is the OpenTelemetry "container.name" attribute.
	Name string
	// ImageName is the OpenTelemetry "container.image.name" attribute.
	ImageName string
	// Runtime is the OpenTelemetry "container.runtime" attribute
	// (e.g. "docker", "containerd").
	Runtime string
}

// IsZero reports whether no container attributes were reported.
func (c Container) IsZero() bool {
	return c.ID == "" && c.Name == "" && c.ImageName == "" && c.Runtime == ""
}
