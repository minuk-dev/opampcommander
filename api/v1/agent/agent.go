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
// It follows the Kubernetes-style resource structure with Metadata, Spec, and Status.
type Agent struct {
	// Metadata contains identifying information about the agent.
	Metadata AgentMetadata `json:"metadata"`

	// Spec contains the desired configuration for the agent.
	Spec AgentSpec `json:"spec"`

	// Status contains the observed state of the agent.
	Status AgentStatus `json:"status"`
} // @name Agent

// AgentMetadata contains identifying information about the agent.
type AgentMetadata struct {
	// InstanceUID is a unique identifier for the agent instance.
	InstanceUID uuid.UUID `json:"instanceUid"`

	// Description is a human-readable description of the agent.
	Description Description `json:"description"`

	// Capabilities is a bitmask representing the capabilities of the agent.
	Capabilities Capabilities `json:"capabilities"`

	// CustomCapabilities is a map of custom capabilities for the agent.
	CustomCapabilities CustomCapabilities `json:"customCapabilities"`
} // @name AgentMetadata

// AgentSpec contains the desired configuration for the agent.
type AgentSpec struct {
	// RemoteConfig is the remote configuration of the agent.
	RemoteConfig RemoteConfig `json:"remoteConfig"`
} // @name AgentSpec

// AgentStatus contains the observed state of the agent.
type AgentStatus struct {
	// EffectiveConfig is the effective configuration of the agent.
	EffectiveConfig EffectiveConfig `json:"effectiveConfig"`

	// PackageStatuses is a map of package statuses for the agent.
	PackageStatuses PackageStatuses `json:"packageStatuses"`

	// ComponentHealth is the health status of the agent's components.
	ComponentHealth ComponentHealth `json:"componentHealth"`

	// AvailableComponents lists components available on the agent.
	AvailableComponents AvailableComponents `json:"availableComponents"`

	// Connected indicates if the agent is currently connected.
	Connected bool `json:"connected"`

	// LastReportedAt is the timestamp when the agent last reported its status.
	LastReportedAt string `json:"lastReportedAt,omitempty"`
} // @name AgentStatus

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
	ConfigMap  map[string]ConfigFile `json:"configMap,omitempty"`
	ConfigHash string                `json:"configHash,omitempty"`
} // @name AgentRemoteConfig

// ComponentHealth represents the health status of the agent's components.
type ComponentHealth struct {
	Healthy       bool              `json:"healthy"`
	StartTimeUnix int64             `json:"startTimeUnix,omitempty"`
	LastError     string            `json:"lastError,omitempty"`
	Status        string            `json:"status,omitempty"`
	StatusTimeMS  int64             `json:"statusTimeMs,omitempty"`
	ComponentsMap map[string]string `json:"componentsMap,omitempty"`
} // @name AgentComponentHealth

// CustomCapabilities represents the custom capabilities of the agent.
type CustomCapabilities struct {
	Capabilities []string `json:"capabilities,omitempty"`
} // @name AgentCustomCapabilities

// AvailableComponents represents the available components of the agent.
type AvailableComponents struct {
	Components map[string]ComponentDetails `json:"components,omitempty"`
} // @name AgentAvailableComponents

// ComponentDetails represents details of an available component.
type ComponentDetails struct {
	Type    string `json:"type,omitempty"`
	Version string `json:"version,omitempty"`
} // @name ComponentDetails

// PackageStatus represents the status of a package in the agent.
type PackageStatus struct {
	// Name is the name of the package.
	Name string `json:"name"`
} // @name AgentPackageStatus
