package v1

const (
	// HostKind is the kind of the host resource.
	HostKind = "Host"
)

// Host represents a machine (bare metal or VM) one or more agents run on.
// Hosts are discovered from the OpenTelemetry attributes agents report, not
// created by users.
type Host struct {
	Kind       string       `json:"kind"`
	APIVersion string       `json:"apiVersion"`
	Metadata   HostMetadata `json:"metadata"`
	Spec       HostSpec     `json:"spec"`
	Status     HostStatus   `json:"status"`
} // @name Host

// HostMetadata contains identity and lifecycle information for a host.
type HostMetadata struct {
	// ID is the stable identity of the host (OpenTelemetry "host.id", falling
	// back to "host.name").
	ID          string            `json:"id"`
	Name        string            `json:"name,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	FirstSeenAt Time              `json:"firstSeenAt"`
	LastSeenAt  Time              `json:"lastSeenAt"`
} // @name HostMetadata

// HostSpec contains the discovered facts about a host.
type HostSpec struct {
	// Platform classifies the deployment environment (baremetal, vm, ...).
	Platform      string `json:"platform"`
	Arch          string `json:"arch,omitempty"`
	Type          string `json:"type,omitempty"`
	OSType        string `json:"osType,omitempty"`
	OSVersion     string `json:"osVersion,omitempty"`
	CloudProvider string `json:"cloudProvider,omitempty"`
	CloudPlatform string `json:"cloudPlatform,omitempty"`
	CloudRegion   string `json:"cloudRegion,omitempty"`
} // @name HostSpec

// HostStatus contains the observed state of a host.
type HostStatus struct {
	// AgentInstanceUIDs are the agents currently associated with this host.
	AgentInstanceUIDs []string    `json:"agentInstanceUids"`
	Conditions        []Condition `json:"conditions,omitempty"`
} // @name HostStatus
