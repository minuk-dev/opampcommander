package v1

import "github.com/google/uuid"

const (
	// AgentKind is the kind of the agent resource.
	AgentKind = "Agent"
)

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
	Description AgentDescription `json:"description"`

	// Capabilities is a bitmask representing the capabilities of the agent.
	Capabilities AgentCapabilities `json:"capabilities"`

	// CustomCapabilities is a map of custom capabilities for the agent.
	CustomCapabilities AgentCustomCapabilities `json:"customCapabilities"`
} // @name AgentMetadata

// AgentSpec contains the desired configuration for the agent.
type AgentSpec struct {
	// NewInstanceUID is a new instance UID to inform the agent of its new identity.
	NewInstanceUID string `json:"newInstanceUid,omitempty"`

	// ConnectionSettings contains connection settings for the agent.
	ConnectionSettings ConnectionSettings `json:"connectionSettings,omitempty"`

	// RemoteConfig is the remote configuration of the agent.
	RemoteConfig AgentRemoteConfig `json:"remoteConfig"`

	// PackagesAvailable is the packages available for the agent to download.
	PackagesAvailable AgentSpecPackages `json:"packagesAvailable,omitempty"`

	// RestartRequiredAt is the time when a restart was requested.
	// If this time is after the agent's start time, the agent should be restarted.
	RestartRequiredAt *Time `json:"restartRequiredAt,omitempty"`
} // @name AgentSpec

// AgentStatus contains the observed state of the agent.
type AgentStatus struct {
	// EffectiveConfig is the effective configuration of the agent.
	EffectiveConfig AgentEffectiveConfig `json:"effectiveConfig"`

	// PackageStatuses is a map of package statuses for the agent.
	PackageStatuses AgentPackageStatuses `json:"packageStatuses"`

	// ComponentHealth is the health status of the agent's components.
	ComponentHealth AgentComponentHealth `json:"componentHealth"`

	// AvailableComponents lists components available on the agent.
	AvailableComponents AgentAvailableComponents `json:"availableComponents"`

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

// AgentCapabilities is a bitmask representing the capabilities of the agent.
type AgentCapabilities uint64

// AgentDescription represents the description of the agent.
type AgentDescription struct {
	// IdentifyingAttributes are attributes that uniquely identify the agent.
	IdentifyingAttributes map[string]string `json:"identifyingAttributes,omitempty"`
	// NonIdentifyingAttributes are attributes that do not uniquely identify the agent.
	NonIdentifyingAttributes map[string]string `json:"nonIdentifyingAttributes,omitempty"`
} // @name AgentDescription

// AgentEffectiveConfig represents the effective configuration of the agent.
type AgentEffectiveConfig struct {
	ConfigMap AgentConfigMap `json:"configMap"`
} // @name AgentEffectiveConfig

// AgentConfigMap represents a map of configuration files for the agent.
type AgentConfigMap struct {
	ConfigMap map[string]AgentConfigFile `json:"configMap"`
} // @name AgentConfigMap

// AgentConfigFile represents a configuration file for the agent.
type AgentConfigFile struct {
	Body        string `json:"body"`
	ContentType string `json:"contentType"`
} // @name AgentConfigFile

// AgentPackageStatuses represents the package statuses of the agent.
type AgentPackageStatuses struct {
	Packages                      map[string]AgentStatusPackageEntry `json:"packages"`
	ServerProvidedAllPackagesHash string                             `json:"serverProvidedAllPackagesHash,omitempty"`
	ErrorMessage                  string                             `json:"errorMessage,omitempty"`
} // @name AgentPackageStatuses

// AgentComponentHealth represents the health status of the agent's components.
type AgentComponentHealth struct {
	Healthy       bool              `json:"healthy"`
	StartTime     Time              `json:"startTime,omitempty"`
	LastError     string            `json:"lastError,omitempty"`
	Status        string            `json:"status,omitempty"`
	StatusTime    Time              `json:"statusTime,omitempty"`
	ComponentsMap map[string]string `json:"componentsMap,omitempty"`
} // @name AgentComponentHealth

// AgentCustomCapabilities represents the custom capabilities of the agent.
type AgentCustomCapabilities struct {
	Capabilities []string `json:"capabilities,omitempty"`
} // @name AgentCustomCapabilities

// AgentAvailableComponents represents the available components of the agent.
type AgentAvailableComponents struct {
	Components map[string]AgentComponentDetails `json:"components,omitempty"`
} // @name AgentAvailableComponents

// AgentComponentDetails represents details of an available component.
type AgentComponentDetails struct {
	Type    string `json:"type,omitempty"`
	Version string `json:"version,omitempty"`
} // @name ComponentDetails

// AgentStatusPackageEntry represents the status of a package in the agent.
type AgentStatusPackageEntry struct {
	// Name is the name of the package.
	Name string `json:"name"`
} // @name AgentPackageStatusPackageEntry

// AgentSpecPackages represents the packages specification for an agent.
type AgentSpecPackages struct {
	// Packages is a list of package names available for the agent.
	Packages []string `json:"packages,omitempty"`
} // @name AgentSpecPackages

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
