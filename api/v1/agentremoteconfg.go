package v1

const (
	// AgentRemoteConfigKind is the kind for AgentRemoteConfig resources.
	AgentRemoteConfigKind = "AgentRemoteConfig"
)

// AgentRemoteConfig represents an agent remote config resource.
type AgentRemoteConfig struct {
	Kind       string                    `json:"kind"`
	APIVersion string                    `json:"apiVersion"`
	Metadata   AgentRemoteConfigMetadata `json:"metadata"`
	Spec       AgentRemoteConfigSpec     `json:"spec"`
	Status     AgentRemoteConfigStatus   `json:"status"`
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
	// SchemaRefs optionally references one or more RemoteConfigSchemas (by name, in
	// the same namespace) this config targets. A config may be compatible with
	// several schemas at once, so this is a list. Referenceable-only.
	SchemaRefs []string `json:"schemaRefs,omitempty"`
}

// AgentRemoteConfigStatus represents the status of an agent remote config.
type AgentRemoteConfigStatus struct {
	Conditions []Condition `json:"conditions"`
} // @name AgentRemoteConfigStatus
