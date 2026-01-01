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

// SetNewInstanceUIDRequest is a struct that represents the request to set a new instance UID for the agent.
type SetNewInstanceUIDRequest struct {
	NewInstanceUID uuid.UUID `binding:"required" json:"newInstanceUid"`
} // @name SetNewInstanceUIDRequest

// Agent represents an agent which is defined OpAMP protocol.
// It follows the Kubernetes-style resource structure with Metadata, Spec, and Status.
type Agent struct {
	// Metadata contains identifying information about the agent.
	Metadata Metadata `json:"metadata"`

	// Spec contains the desired configuration for the agent.
	Spec Spec `json:"spec"`

	// Status contains the observed state of the agent.
	Status Status `json:"status"`
} // @name Agent

// Metadata contains identifying information about the agent.
type Metadata struct {
	// InstanceUID is a unique identifier for the agent instance.
	InstanceUID uuid.UUID `json:"instanceUid"`

	// Description is a human-readable description of the agent.
	Description Description `json:"description"`

	// Capabilities is a bitmask representing the capabilities of the agent.
	Capabilities Capabilities `json:"capabilities"`

	// CustomCapabilities is a map of custom capabilities for the agent.
	CustomCapabilities CustomCapabilities `json:"customCapabilities"`
} // @name AgentMetadata

// Spec contains the desired configuration for the agent.
type Spec struct {
	// NewInstanceUID is a new instance UID to inform the agent of its new identity.
	NewInstanceUID string `json:"newInstanceUid,omitempty"`

	// ConnectionSettings contains connection settings for the agent.
	ConnectionSettings ConnectionSettings `json:"connectionSettings,omitempty"`

	// RemoteConfig is the remote configuration of the agent.
	RemoteConfig RemoteConfig `json:"remoteConfig"`
} // @name AgentSpec

// Status contains the observed state of the agent.
type Status struct {
	// EffectiveConfig is the effective configuration of the agent.
	EffectiveConfig EffectiveConfig `json:"effectiveConfig"`

	// PackageStatuses is a map of package statuses for the agent.
	PackageStatuses PackageStatuses `json:"packageStatuses"`

	// ComponentHealth is the health status of the agent's components.
	ComponentHealth ComponentHealth `json:"componentHealth"`

	// AvailableComponents lists components available on the agent.
	AvailableComponents AvailableComponents `json:"availableComponents"`

	// Conditions is a list of conditions that apply to the agent.
	Conditions []Condition `json:"conditions"`

	// Connected indicates if the agent is currently connected.
	Connected bool `json:"connected"`

	// ConnectionType indicates the type of connection the agent is using.
	ConnectionType string `json:"connectionType,omitempty"`

	// SequenceNum is the sequence number from the last AgentToServer message.
	SequenceNum uint64 `json:"sequenceNum,omitempty"`

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

// Condition represents a condition of an agent.
type Condition struct {
	Type               ConditionType   `json:"type"`
	LastTransitionTime string          `json:"lastTransitionTime"`
	Status             ConditionStatus `json:"status"`
	Reason             string          `json:"reason"`
	Message            string          `json:"message,omitempty"`
} // @name AgentCondition

// ConditionType represents the type of an agent condition.
type ConditionType string // @name AgentConditionType

const (
	// ConditionTypeConnected represents the condition when the agent is connected.
	ConditionTypeConnected ConditionType = "Connected"
	// ConditionTypeHealthy represents the condition when the agent is healthy.
	ConditionTypeHealthy ConditionType = "Healthy"
	// ConditionTypeConfigured represents the condition when the agent has been configured.
	ConditionTypeConfigured ConditionType = "Configured"
	// ConditionTypeRegistered represents the condition when the agent has been registered.
	ConditionTypeRegistered ConditionType = "Registered"
)

// ConditionStatus represents the status of an agent condition.
type ConditionStatus string // @name AgentConditionStatus

const (
	// ConditionStatusTrue represents a true condition status.
	ConditionStatusTrue ConditionStatus = "True"
	// ConditionStatusFalse represents a false condition status.
	ConditionStatusFalse ConditionStatus = "False"
	// ConditionStatusUnknown represents an unknown condition status.
	ConditionStatusUnknown ConditionStatus = "Unknown"
)

// ConnectionSettings represents connection settings for the agent.
type ConnectionSettings struct {
	// OpAMP contains OpAMP server connection settings.
	OpAMP OpAMPConnectionSettings `json:"opamp,omitempty"`
	// OwnMetrics contains own metrics connection settings.
	OwnMetrics TelemetryConnectionSettings `json:"ownMetrics,omitempty"`
	// OwnLogs contains own logs connection settings.
	OwnLogs TelemetryConnectionSettings `json:"ownLogs,omitempty"`
	// OwnTraces contains own traces connection settings.
	OwnTraces TelemetryConnectionSettings `json:"ownTraces,omitempty"`
	// OtherConnections contains other connection settings mapped by name.
	OtherConnections map[string]OtherConnectionSettings `json:"otherConnections,omitempty"`
} // @name ConnectionSettings

// OpAMPConnectionSettings represents OpAMP connection settings.
type OpAMPConnectionSettings struct {
	// DestinationEndpoint is the URL to connect to the OpAMP server.
	DestinationEndpoint string `json:"destinationEndpoint"`
	// Headers are HTTP headers to include in requests.
	Headers map[string][]string `json:"headers,omitempty"`
	// Certificate contains TLS certificate information.
	Certificate TLSCertificate `json:"certificate,omitempty"`
} // @name OpAMPConnectionSettings

// TelemetryConnectionSettings represents telemetry connection settings.
type TelemetryConnectionSettings struct {
	// DestinationEndpoint is the URL to send telemetry data to.
	DestinationEndpoint string `json:"destinationEndpoint"`
	// Headers are HTTP headers to include in requests.
	Headers map[string][]string `json:"headers,omitempty"`
	// Certificate contains TLS certificate information.
	Certificate TLSCertificate `json:"certificate,omitempty"`
} // @name TelemetryConnectionSettings

// OtherConnectionSettings represents other connection settings.
type OtherConnectionSettings struct {
	// DestinationEndpoint is the URL to connect to.
	DestinationEndpoint string `json:"destinationEndpoint"`
	// Headers are HTTP headers to include in requests.
	Headers map[string][]string `json:"headers,omitempty"`
	// Certificate contains TLS certificate information.
	Certificate TLSCertificate `json:"certificate,omitempty"`
} // @name OtherConnectionSettings

// TLSCertificate represents TLS certificate information.
type TLSCertificate struct {
	// Cert is the PEM-encoded certificate.
	Cert string `json:"cert,omitempty"`
	// PrivateKey is the PEM-encoded private key.
	PrivateKey string `json:"privateKey,omitempty"`
	// CaCert is the PEM-encoded CA certificate.
	CaCert string `json:"caCert,omitempty"`
} // @name TLSCertificate

// SetConnectionSettingsRequest represents a request to set connection settings for an agent.
type SetConnectionSettingsRequest struct {
	// ConnectionSettings contains the connection settings to set.
	ConnectionSettings ConnectionSettings `binding:"required" json:"connectionSettings"`
} // @name SetConnectionSettingsRequest
