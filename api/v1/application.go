package v1

// ApplicationKind is the kind of an application resource.
const ApplicationKind = "Application"

// Application is a discovered logical service, not a user-created resource.
type Application struct {
	Kind       string              `json:"kind"`
	APIVersion string              `json:"apiVersion"`
	Metadata   ApplicationMetadata `json:"metadata"`
	Spec       ApplicationSpec     `json:"spec"`
	Status     ApplicationStatus   `json:"status"`
} // @name Application

// ApplicationMetadata contains application identity and lifecycle information.
type ApplicationMetadata struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	FirstSeenAt Time              `json:"firstSeenAt"`
	LastSeenAt  Time              `json:"lastSeenAt"`
} // @name ApplicationMetadata

// ApplicationSpec contains observed service attributes.
type ApplicationSpec struct {
	Namespace string   `json:"namespace,omitempty"`
	Name      string   `json:"name"`
	Versions  []string `json:"versions,omitempty"`
	AgentType string   `json:"agentType,omitempty"`
} // @name ApplicationSpec

// ApplicationStatus contains observed application state.
type ApplicationStatus struct {
	AgentInstanceUIDs []string    `json:"agentInstanceUids"`
	Conditions        []Condition `json:"conditions,omitempty"`
} // @name ApplicationStatus
