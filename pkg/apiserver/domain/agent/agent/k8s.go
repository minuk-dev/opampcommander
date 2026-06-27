package agent

// K8s is a descriptor value object that represents the Kubernetes context an
// agent runs in. Its fields are derived from the OpenTelemetry "k8s.*"
// non-identifying resource attributes reported by the agent.
type K8s struct {
	// PodName is the OpenTelemetry "k8s.pod.name" attribute.
	PodName string
	// PodUID is the OpenTelemetry "k8s.pod.uid" attribute. It is stable for the
	// lifetime of a pod and is the preferred identity for a containerized agent.
	PodUID string
	// NamespaceName is the OpenTelemetry "k8s.namespace.name" attribute.
	NamespaceName string
	// NodeName is the OpenTelemetry "k8s.node.name" attribute. It links a pod to
	// the Host (node) it runs on.
	NodeName string
	// ContainerName is the OpenTelemetry "k8s.container.name" attribute.
	ContainerName string
}

// IsZero reports whether no Kubernetes attributes were reported.
func (k K8s) IsZero() bool {
	return k.PodName == "" && k.PodUID == "" && k.NamespaceName == "" &&
		k.NodeName == "" && k.ContainerName == ""
}
