package v1

// AgentRemoteConfig represents an agent remote config resource.
type AgentRemoteConfig struct {
	Metadata AgentRemoteConfigMetadata `json:"metadata"`
	Spec     AgentRemoteConfigSpec     `json:"spec"`
	Status   AgentRemoteConfigStatus   `json:"status"`
} // @name AgentRemoteConfig

// AgentRemoteConfigMetadata represents the metadata of an agent remote config.
type AgentRemoteConfigMetadata struct {
	Name       string     `json:"name"`
	Attributes Attributes `json:"attributes"`
	CreatedAt  *Time      `json:"createdAt,omitempty"`
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
