// Package agent provides the agent API for the server
package agent

import "github.com/google/uuid"

const (
	// AgentKind is the kind of the agent resource.
	AgentKind = "Agent"
)

// UpdateAgentConfigRequest is a struct that represents the request to update the agent configuration.
// It contains the target instance UID and the remote configuration data.
type UpdateAgentConfigRequest struct {
	RemoteConfig any `binding:"required" json:"remoteConfig"`
} // @name UpdateAgentConfigRequest

// Agent represents an agent which is defined OpAMP protocol.
// It is a value object that contains the instance UID and raw data.
type Agent struct {
	// InstanceUID is a unique identifier for the agent instance.
	InstanceUID uuid.UUID `json:"instanceUid"`

	// IsManaged indicates whether the agent is managed by the server.
	// If true, the server manages the agent and can send commands to it.
	IsManaged bool `json:"isManaged"`

	// Capabilities is a bitmask representing the capabilities of the agent.
	// It is used to determine what features the agent supports.
	// If nil, it means the capabilities are unspecified.
	Capabilities Capabilities `json:"capabilities"`

	// Description is a human-readable description of the agent.
	Description Description `json:"description"`

	// EffectiveConfig is the effective configuration of the agent.
	// It is used to determine the current configuration of the agent.
	EffectiveConfig EffectiveConfig `json:"effectiveConfig"`

	// PackageStatuses is a map of package statuses for the agent.
	PackageStatuses PackageStatuses `json:"packageStatuses"`

	// ComponentHealth is the health status of the agent's components.
	ComponentHealth ComponentHealth `json:"componentHealth"`

	// RemoteConfig is the remote configuration of the agent.
	// It is used to determine the current remote configuration of the agent.
	RemoteConfig RemoteConfig `json:"remoteConfig"`

	// CustomCapabilities is a map of custom capabilities for the agent.
	CustomCapabilities CustomCapabilities `json:"customCapabilities"`

	AvailableComponents AvailableComponents `json:"availableComponents"`
} // @name Agent

// Capabilities is a bitmask representing the capabilities of the agent.
type Capabilities uint64

// Description represents the description of the agent.
type Description struct {
	// IdentifyingAttributes are attributes that uniquely identify the agent.
	IdentifyingAttributes map[string]string `json:"identifyingAttributes,omitempty"`
	// NonIdentifyingAttributes are attributes that do not uniquely identify the agent.
	NonIdentifyingAttributes map[string]string `json:"nonIdentifyingAttributes,omitempty"`
} // @name AgentDescription

// EffectiveConfig represents the effective configuration of the agent.
type EffectiveConfig struct {
	ConfigMap ConfigMap `json:"configMap"`
} // @name AgentEffectiveConfig

// ConfigMap represents a map of configuration files for the agent.
type ConfigMap struct {
	ConfigMap map[string]ConfigFile `json:"configMap"`
} // @name AgentConfigMap

// ConfigFile represents a configuration file for the agent.
type ConfigFile struct {
	Body        string `json:"body"`
	ContentType string `json:"contentType"`
} // @name AgentConfigFile

// PackageStatuses represents the package statuses of the agent.
type PackageStatuses struct {
	Packages                      map[string]PackageStatus `json:"packages"`
	ServerProvidedAllPackagesHash string                   `json:"serverProvidedAllPackagesHash,omitempty"`
	ErrorMessage                  string                   `json:"errorMessage,omitempty"`
} // @name AgentPackageStatuses

// RemoteConfig represents the remote configuration of the agent.
type RemoteConfig struct {
} // @name AgentRemoteConfig

// ComponentHealth represents the health status of the agent's components.
type ComponentHealth struct {
} // @name AgentComponentHealth

// CustomCapabilities represents the custom capabilities of the agent.
type CustomCapabilities struct {
} // @name AgentCustomCapabilities

// AvailableComponents represents the available components of the agent.
type AvailableComponents struct {
} // @name AgentAvailableComponents

// PackageStatus represents the status of a package in the agent.
type PackageStatus struct {
	// Name is the name of the package.
	Name string `json:"name"`
} // @name AgentPackageStatus
