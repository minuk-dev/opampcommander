package v1

const (
	// AgentRemoteConfigKind is the kind for AgentRemoteConfig resources.
	AgentRemoteConfigKind = "AgentRemoteConfig"
)

// AgentRemoteConfig represents an agent remote config resource.
type AgentRemoteConfig struct {
	Metadata AgentRemoteConfigMetadata `json:"metadata"`
	Spec     AgentRemoteConfigSpec     `json:"spec"`
	Status   AgentRemoteConfigStatus   `json:"status"`
} // @name AgentRemoteConfig

// AgentRemoteConfigMetadata represents the metadata of an agent remote config.
type AgentRemoteConfigMetadata struct {
	Name       string     `json:"name"`
	Namespace  string     `json:"namespace"`
	Attributes Attributes `json:"attributes"`
	CreatedAt  Time       `json:"createdAt"`
} // @name AgentRemoteConfigMetadata

// AgentRemoteConfigSpec represents the specification of an agent remote config.
type AgentRemoteConfigSpec struct {
	Value       string `json:"value"`
	ContentType string `json:"contentType"`
}

// AgentRemoteConfigStatus represents the status of an agent remote config.
type AgentRemoteConfigStatus struct {
	Conditions []Condition `json:"conditions"`
} // @name AgentRemoteConfigStatus
