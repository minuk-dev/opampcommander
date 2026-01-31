package v1

type AgentRemoteConfig struct {
	Metadata AgentRemoteConfigMetadata `json:"metadata"`
	Spec     AgentRemoteConfigSpec     `json:"spec"`
	Status   AgentRemoteConfigStatus   `json:"status"`
} // @name AgentRemoteConfig

type AgentRemoteConfigMetadata struct {
	Name       string     `json:"name"`
	Attributes Attributes `json:"attributes"`
} // @name AgentRemoteConfigMetadata

type AgentRemoteConfigSpec struct {
	Value       string `json:"value"`
	ContentType string `json:"contentType"`
}

type AgentRemoteConfigStatus struct {
	Conditions []Condition `json:"conditions"`
} // @name AgentRemoteConfigStatus
