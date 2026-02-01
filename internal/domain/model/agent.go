package model

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/minuk-dev/opampcommander/internal/domain/model/agent"
	"github.com/minuk-dev/opampcommander/internal/domain/model/vo"
)

var (
	// ErrUnsupportedRemoteConfigContentType is returned when the remote config content type is not supported.
	ErrUnsupportedRemoteConfigContentType = errors.New("unsupported remote config content type")
	// ErrUnsupportedAgentOperation is returned when the agent does not support the requested operation.
	ErrUnsupportedAgentOperation = errors.New("unsupported agent operation")
)

// Agent is a domain model to control opamp agent by opampcommander.
type Agent struct {
	Metadata AgentMetadata
	Spec     AgentSpec
	Status   AgentStatus
}

// NewAgent creates a new agent with the given instance UID.
// It initializes all fields with default values.
// You can optionally pass AgentOption functions to customize the agent.
//
//nolint:funlen
func NewAgent(instanceUID uuid.UUID, opts ...AgentOption) *Agent {
	agent := &Agent{
		//exhaustruct:ignore
		Metadata: AgentMetadata{
			InstanceUID: instanceUID,
		},
		//exhaustruct:ignore
		Spec: AgentSpec{
			NewInstanceUID:    uuid.Nil,
			RestartInfo:       nil,
			RemoteConfig:      nil,
			ConnectionInfo:    nil,
			PackagesAvailable: nil,
		},
		Status: AgentStatus{
			RemoteConfigStatus: AgentRemoteConfigStatus{
				LastRemoteConfigHash: nil,
				Status:               RemoteConfigStatusUnset,
				ErrorMessage:         "",
				LastUpdatedAt:        time.Time{},
			},
			ConnectionSettingsStatus: AgentConnectionSettingsStatus{
				LastConnectionSettingsHash: nil,
				Status:                     ConnectionSettingsStatusUnset,
				ErrorMessage:               "",
			},
			EffectiveConfig: AgentEffectiveConfig{
				ConfigMap: AgentConfigMap{
					ConfigMap: make(map[string]AgentConfigFile),
				},
			},
			//exhaustruct:ignore
			PackageStatuses: AgentPackageStatuses{
				Packages: make(map[string]AgentPackageStatusEntry),
			},
			//exhaustruct:ignore
			ComponentHealth: AgentComponentHealth{
				StartTime:          time.Now(),
				ComponentHealthMap: make(map[string]AgentComponentHealth),
			},
			//exhaustruct:ignore
			AvailableComponents: AgentAvailableComponents{
				Components: make(map[string]ComponentDetails),
			},
			Conditions: []AgentCondition{
				{
					Type:               AgentConditionTypeRegistered,
					LastTransitionTime: time.Now(),
					Status:             AgentConditionStatusTrue,
					Reason:             "system",
					Message:            "Agent registered",
				},
			},
			Connected:      false,
			ConnectionType: ConnectionTypeUnknown,
			SequenceNum:    0,
			LastReportedAt: time.Time{},
			LastReportedTo: "",
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(agent)
	}

	return agent
}

// ApplyRemoteConfig applies a remote config to the agent.
func (a *Agent) ApplyRemoteConfig(agentRemoteConfigName string) error {
	if a.Spec.RemoteConfig == nil {
		a.Spec.RemoteConfig = &AgentSpecRemoteConfig{}
	}

	a.Spec.RemoteConfig.RemoteConfigNames = append(a.Spec.RemoteConfig.RemoteConfigNames, agentRemoteConfigName)
	sort.Strings(a.Spec.RemoteConfig.RemoteConfigNames)
	a.Spec.RemoteConfig.RemoteConfigNames = lo.Uniq(a.Spec.RemoteConfig.RemoteConfigNames)

	return nil
}

// IsConnected checks if the agent is currently connected.
func (a *Agent) IsConnected(_ context.Context) bool {
	return !a.Status.LastReportedAt.IsZero()
}

// NeedFullStateCommand checks if the agent needs to send a ReportFullState command.
func (a *Agent) NeedFullStateCommand() bool {
	return !a.HasInstanceUID() || !a.Metadata.IsComplete()
}

// HasPendingServerMessages checks if there are any pending server messages for the agent.
func (a *Agent) HasPendingServerMessages() bool {
	return a.NeedFullStateCommand() ||
		a.HasRemoteConfig() ||
		a.ShouldBeRestarted()
}

// HasInstanceUID checks if the agent has a valid instance UID.
func (a *Agent) HasInstanceUID() bool {
	return a.Spec.NewInstanceUID != uuid.Nil
}

// ShouldBeRestarted checks if the agent should be restarted to apply a command that requires a restart.
func (a *Agent) ShouldBeRestarted() bool {
	if a.Spec.RestartInfo == nil {
		return false
	}

	return a.Spec.RestartInfo.ShouldBeRestarted(a.Status.ComponentHealth.StartTime)
}

// ShouldBeRestarted checks if the agent should be restarted to apply a command that requires a restart.
func (a *AgentRestartInfo) ShouldBeRestarted(agentStartTime time.Time) bool {
	if a == nil {
		return false
	}

	return !a.RequiredRestartedAt.IsZero() &&
		a.RequiredRestartedAt.After(agentStartTime)
}

// IsRestartSupported checks if the agent supports restart command.
func (a *Agent) IsRestartSupported() bool {
	return a.Metadata.Capabilities.HasAcceptsRestartCommand()
}

// SetRestartRequired sets the restart required information for the agent.
func (a *Agent) SetRestartRequired(requiredAt time.Time) error {
	if !a.IsRestartSupported() {
		return ErrUnsupportedAgentOperation
	}

	if a.Spec.RestartInfo == nil {
		a.Spec.RestartInfo = &AgentRestartInfo{}
	}

	a.Spec.RestartInfo.RequiredRestartedAt = requiredAt

	return nil
}

// ConnectedServerID returns the server the agent is currently connected to.
func (a *Agent) ConnectedServerID() (string, error) {
	return a.Status.LastReportedTo, nil
}

// AgentOption is a function that configures an Agent.
type AgentOption func(*Agent)

// WithDescription sets the agent description.
func WithDescription(description *agent.Description) AgentOption {
	return func(a *Agent) {
		if description != nil {
			a.Metadata.Description = *description
		}
	}
}

// WithCapabilities sets the agent capabilities.
func WithCapabilities(capabilities *agent.Capabilities) AgentOption {
	return func(a *Agent) {
		if capabilities != nil {
			a.Metadata.Capabilities = *capabilities
		}
	}
}

// WithCustomCapabilities sets the agent custom capabilities.
func WithCustomCapabilities(customCapabilities *AgentCustomCapabilities) AgentOption {
	return func(a *Agent) {
		if customCapabilities != nil {
			a.Metadata.CustomCapabilities = *customCapabilities
		}
	}
}

// WithEffectiveConfig sets the agent effective config.
func WithEffectiveConfig(effectiveConfig *AgentEffectiveConfig) AgentOption {
	return func(a *Agent) {
		if effectiveConfig != nil {
			a.Status.EffectiveConfig = *effectiveConfig
		}
	}
}

// WithComponentHealth sets the agent component health.
func WithComponentHealth(componentHealth *AgentComponentHealth) AgentOption {
	return func(a *Agent) {
		if componentHealth != nil {
			a.Status.ComponentHealth = *componentHealth
		}
	}
}

// WithPackageStatuses sets the agent package statuses.
func WithPackageStatuses(packageStatuses *AgentPackageStatuses) AgentOption {
	return func(a *Agent) {
		if packageStatuses != nil {
			a.Status.PackageStatuses = *packageStatuses
		}
	}
}

// WithAvailableComponents sets the agent available components.
func WithAvailableComponents(availableComponents *AgentAvailableComponents) AgentOption {
	return func(a *Agent) {
		if availableComponents != nil {
			a.Status.AvailableComponents = *availableComponents
		}
	}
}

// AgentMetadata is a domain model to control opamp agent metadata.
type AgentMetadata struct {
	// InstanceUID is a unique identifier for the agent instance.
	// It is generated by the agent and should not change between restarts of the agent.
	InstanceUID uuid.UUID

	// Description is a agent description defined in the opamp protocol.
	// It is set by the agent and should not change between restarts of the agent.
	// It can be changed by the agent at any time.
	Description agent.Description

	// Capabilities is a agent capabilities defined in the opamp protocol.
	Capabilities agent.Capabilities

	// CustomCapabilities is a list of custom capabilities that the Agent supports.
	CustomCapabilities AgentCustomCapabilities
}

// IsComplete checks if all required metadata fields are populated.
// Returns true if the agent has reported its description and capabilities.
func (am *AgentMetadata) IsComplete() bool {
	// Check if Description has any attributes
	hasDescription := len(am.Description.IdentifyingAttributes) > 0 ||
		len(am.Description.NonIdentifyingAttributes) > 0

	// Check if Capabilities is not zero (unset)
	hasCapabilities := am.Capabilities != 0

	return hasDescription && hasCapabilities
}

// AgentStatus is a domain model to control opamp agent status.
type AgentStatus struct {
	RemoteConfigStatus       AgentRemoteConfigStatus
	ConnectionSettingsStatus AgentConnectionSettingsStatus
	EffectiveConfig          AgentEffectiveConfig
	PackageStatuses          AgentPackageStatuses
	ComponentHealth          AgentComponentHealth
	AvailableComponents      AgentAvailableComponents

	// Conditions is a list of conditions that apply to the agent.
	Conditions []AgentCondition

	Connected      bool
	ConnectionType ConnectionType

	SequenceNum    uint64
	LastReportedAt time.Time
	// LastReportedTo is the ID of the server the agent last reported to.
	// When you want to get Server object, use `GetServerByID` function from ServerUsecase.
	LastReportedTo string
}

// AgentCondition represents a condition of an agent.
type AgentCondition struct {
	// Type is the type of the condition.
	Type AgentConditionType
	// LastTransitionTime is the last time the condition transitioned.
	LastTransitionTime time.Time
	// Status is the status of the condition.
	Status AgentConditionStatus
	// Reason is the identifier of the user or system that triggered the condition.
	Reason string
	// Message is a human readable message indicating details about the condition.
	Message string
}

// AgentConditionType represents the type of an agent condition.
type AgentConditionType string

const (
	// AgentConditionTypeConnected represents the condition when the agent is connected.
	AgentConditionTypeConnected AgentConditionType = "Connected"
	// AgentConditionTypeHealthy represents the condition when the agent is healthy.
	AgentConditionTypeHealthy AgentConditionType = "Healthy"
	// AgentConditionTypeConfigured represents the condition when the agent has been configured.
	AgentConditionTypeConfigured AgentConditionType = "Configured"
	// AgentConditionTypeRegistered represents the condition when the agent has been registered.
	AgentConditionTypeRegistered AgentConditionType = "Registered"
)

// AgentConditionStatus represents the status of an agent condition.
type AgentConditionStatus string

const (
	// AgentConditionStatusTrue represents a true condition status.
	AgentConditionStatusTrue AgentConditionStatus = "True"
	// AgentConditionStatusFalse represents a false condition status.
	AgentConditionStatusFalse AgentConditionStatus = "False"
	// AgentConditionStatusUnknown represents an unknown condition status.
	AgentConditionStatusUnknown AgentConditionStatus = "Unknown"
)

// AgentSpec is a domain model to control opamp agent spec.
type AgentSpec struct {
	// NewInstanceUID is a new instance UID to inform the agent of its new identity.
	NewInstanceUID uuid.UUID

	// RestartInfo contains information about agent restart.
	RestartInfo *AgentRestartInfo

	// ConnectionInfo is the connection information for the agent.
	ConnectionInfo *ConnectionInfo

	// RemoteConfig is the remote configuration for the agent.
	RemoteConfig *AgentSpecRemoteConfig

	// PackagesAvailable is the packages available for the agent.
	PackagesAvailable *AgentSpecPackage
}

// AgentSpecRemoteConfig represents the remote config specification for an agent.
type AgentSpecRemoteConfig struct {
	RemoteConfigNames []string
}

// AgentSpecPackage represents the package specification for an agent.
type AgentSpecPackage struct {
	// Packages is a list of package names available for the agent.
	Packages []string
}

// Hash computes the hash of the agent spec packages.
func (a *AgentSpecPackage) Hash() (vo.Hash, error) {
	data, err := json.Marshal(a.Packages)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal agent spec packages for hashing: %w", err)
	}

	hash, err := vo.NewHash(data)
	if err != nil {
		return nil, fmt.Errorf("failed to create hash for agent spec packages: %w", err)
	}

	return hash, nil
}

type connectionSettings struct {
	headers     map[string][]string
	certificate TelemetryTLSCertificate
}

// ConnectionOption is a function option for configuring connection settings.
type ConnectionOption interface {
	apply(cs *connectionSettings)
}

// ConnectionOptionFunc is a function that implements ConnectionOption.
type ConnectionOptionFunc func(*connectionSettings)

func (f ConnectionOptionFunc) apply(cs *connectionSettings) {
	f(cs)
}

// WithHeaders sets the headers for the connection.
func WithHeaders(headers map[string][]string) ConnectionOption {
	return ConnectionOptionFunc(func(cs *connectionSettings) {
		cs.headers = headers
	})
}

// WithCertificate sets the certificate for the connection.
func WithCertificate(certificate TelemetryTLSCertificate) ConnectionOption {
	return ConnectionOptionFunc(func(cs *connectionSettings) {
		cs.certificate = certificate
	})
}

// SetOpAMPConnectionSettings sets OpAMP connection settings for the agent.
func (a *Agent) SetOpAMPConnectionSettings(endpoint string, opts ...ConnectionOption) error {
	//exhaustruct:ignore
	settings := &connectionSettings{}
	for _, opt := range opts {
		opt.apply(settings)
	}

	err := a.Spec.ConnectionInfo.SetOpAMP(OpAMPConnectionSettings{
		DestinationEndpoint: endpoint,
		Headers:             settings.headers,
		Certificate:         settings.certificate,
	})
	if err != nil {
		return fmt.Errorf("failed to set OpAMP connection settings: %w", err)
	}

	return nil
}

// SetMetricsConnectionSettings sets metrics connection settings for the agent.
func (a *Agent) SetMetricsConnectionSettings(endpoint string, opts ...ConnectionOption) error {
	//exhaustruct:ignore
	settings := &connectionSettings{}
	for _, opt := range opts {
		opt.apply(settings)
	}

	err := a.Spec.ConnectionInfo.SetOwnMetrics(TelemetryConnectionSettings{
		DestinationEndpoint: endpoint,
		Headers:             settings.headers,
		Certificate:         settings.certificate,
	})
	if err != nil {
		return fmt.Errorf("failed to set metrics connection settings: %w", err)
	}

	return nil
}

// SetLogsConnectionSettings sets logs connection settings for the agent.
func (a *Agent) SetLogsConnectionSettings(endpoint string, opts ...ConnectionOption) error {
	//exhaustruct:ignore
	settings := &connectionSettings{}
	for _, opt := range opts {
		opt.apply(settings)
	}

	err := a.Spec.ConnectionInfo.SetOwnLogs(TelemetryConnectionSettings{
		DestinationEndpoint: endpoint,
		Headers:             settings.headers,
		Certificate:         settings.certificate,
	})
	if err != nil {
		return fmt.Errorf("failed to set logs connection settings: %w", err)
	}

	return nil
}

// SetTracesConnectionSettings sets traces connection settings for the agent.
func (a *Agent) SetTracesConnectionSettings(endpoint string, opts ...ConnectionOption) error {
	//exhaustruct:ignore
	settings := &connectionSettings{}
	for _, opt := range opts {
		opt.apply(settings)
	}

	err := a.Spec.ConnectionInfo.SetOwnTraces(TelemetryConnectionSettings{
		DestinationEndpoint: endpoint,
		Headers:             settings.headers,
		Certificate:         settings.certificate,
	})
	if err != nil {
		return fmt.Errorf("failed to set traces connection settings: %w", err)
	}

	return nil
}

// SetOtherConnectionSettings sets other connection settings for the agent.
func (a *Agent) SetOtherConnectionSettings(name, endpoint string, opts ...ConnectionOption) error {
	//exhaustruct:ignore
	settings := &connectionSettings{}
	for _, opt := range opts {
		opt.apply(settings)
	}

	existingConnections := a.Spec.ConnectionInfo.OtherConnections()
	existingConnections[name] = OtherConnectionSettings{
		DestinationEndpoint: endpoint,
		Headers:             settings.headers,
		Certificate:         settings.certificate,
	}

	err := a.Spec.ConnectionInfo.SetOtherConnection(name, existingConnections[name])
	if err != nil {
		return fmt.Errorf("failed to set other connection settings: %w", err)
	}

	return nil
}

// IsOpAMPConnectionSettingsSupported checks if the agent supports OpAMP connection settings.
func (a *Agent) IsOpAMPConnectionSettingsSupported() bool {
	return a.Metadata.Capabilities.HasAcceptsOpAMPConnectionSettings()
}

// IsOwnMetricsSupported checks if the agent supports reporting its own metrics.
func (a *Agent) IsOwnMetricsSupported() bool {
	return a.Metadata.Capabilities.HasReportsOwnTraces()
}

// IsOwnLogsSupported checks if the agent supports reporting its own logs.
func (a *Agent) IsOwnLogsSupported() bool {
	return a.Metadata.Capabilities.HasReportsOwnLogs()
}

// IsOwnTracesSupported checks if the agent supports reporting its own traces.
func (a *Agent) IsOwnTracesSupported() bool {
	return a.Metadata.Capabilities.HasReportsOwnMetrics()
}

// IsOtherConnectionSettingsSupported checks if the agent supports other connection settings.
func (a *Agent) IsOtherConnectionSettingsSupported() bool {
	return a.Metadata.Capabilities.HasAcceptsOpAMPConnectionSettings()
}

// ApplyConnectionSettings applies connection settings to the agent from agent group.
func (a *Agent) ApplyConnectionSettings(
	opamp OpAMPConnectionSettings,
	ownMetrics TelemetryConnectionSettings,
	ownLogs TelemetryConnectionSettings,
	ownTraces TelemetryConnectionSettings,
	otherConnections map[string]OtherConnectionSettings,
) error {
	connectionInfo, err := NewConnectionInfo(opamp, ownMetrics, ownLogs, ownTraces, otherConnections)
	if err != nil {
		return fmt.Errorf("failed to create connection info: %w", err)
	}

	a.Spec.ConnectionInfo = connectionInfo

	return nil
}

// ConnectionInfo represents connection information for the agent.
type ConnectionInfo struct {
	Hash vo.Hash

	opamp            OpAMPConnectionSettings
	ownMetrics       TelemetryConnectionSettings
	ownLogs          TelemetryConnectionSettings
	ownTraces        TelemetryConnectionSettings
	otherConnections map[string]OtherConnectionSettings
}

// NewConnectionInfo creates a new ConnectionInfo with the given settings.
func NewConnectionInfo(
	opamp OpAMPConnectionSettings,
	ownMetrics TelemetryConnectionSettings,
	ownLogs TelemetryConnectionSettings,
	ownTraces TelemetryConnectionSettings,
	otherConnections map[string]OtherConnectionSettings,
) (*ConnectionInfo, error) {
	connectionInfo := &ConnectionInfo{
		Hash:             nil,
		opamp:            opamp,
		ownMetrics:       ownMetrics,
		ownLogs:          ownLogs,
		ownTraces:        ownTraces,
		otherConnections: otherConnections,
	}

	err := connectionInfo.updateHash()
	if err != nil {
		return nil, fmt.Errorf("failed to create connection info: %w", err)
	}

	return connectionInfo, nil
}

// OpAMP returns the OpAMP connection settings.
func (ci *ConnectionInfo) OpAMP() OpAMPConnectionSettings {
	return ci.opamp
}

// OwnMetrics returns the own metrics connection settings.
func (ci *ConnectionInfo) OwnMetrics() TelemetryConnectionSettings {
	return ci.ownMetrics
}

// OwnLogs returns the own logs connection settings.
func (ci *ConnectionInfo) OwnLogs() TelemetryConnectionSettings {
	return ci.ownLogs
}

// OwnTraces returns the own traces connection settings.
func (ci *ConnectionInfo) OwnTraces() TelemetryConnectionSettings {
	return ci.ownTraces
}

// OtherConnections returns the other connection settings.
func (ci *ConnectionInfo) OtherConnections() map[string]OtherConnectionSettings {
	ret := maps.Clone(ci.otherConnections)
	if ret == nil {
		ret = make(map[string]OtherConnectionSettings)
	}

	return ret
}

// UpdateAllConnections updates all connection settings.
func (ci *ConnectionInfo) UpdateAllConnections(
	opamp OpAMPConnectionSettings,
	ownMetrics TelemetryConnectionSettings,
	ownLogs TelemetryConnectionSettings,
	ownTraces TelemetryConnectionSettings,
	otherConnections map[string]OtherConnectionSettings,
) error {
	ci.opamp = opamp
	ci.ownMetrics = ownMetrics
	ci.ownLogs = ownLogs
	ci.ownTraces = ownTraces
	ci.otherConnections = otherConnections

	return ci.updateHash()
}

// SetOpAMP sets the OpAMP connection settings.
func (ci *ConnectionInfo) SetOpAMP(settings OpAMPConnectionSettings) error {
	ci.opamp = settings

	return ci.updateHash()
}

// SetOwnMetrics sets the own metrics connection settings.
func (ci *ConnectionInfo) SetOwnMetrics(settings TelemetryConnectionSettings) error {
	ci.ownMetrics = settings

	return ci.updateHash()
}

// SetOwnLogs sets the own logs connection settings.
func (ci *ConnectionInfo) SetOwnLogs(settings TelemetryConnectionSettings) error {
	ci.ownLogs = settings

	return ci.updateHash()
}

// SetOwnTraces sets the own traces connection settings.
func (ci *ConnectionInfo) SetOwnTraces(settings TelemetryConnectionSettings) error {
	ci.ownTraces = settings

	return ci.updateHash()
}

// SetOtherConnection sets the other connection settings.
func (ci *ConnectionInfo) SetOtherConnection(name string, settings OtherConnectionSettings) error {
	if ci.otherConnections == nil {
		ci.otherConnections = make(map[string]OtherConnectionSettings)
	}

	ci.otherConnections[name] = settings

	return ci.updateHash()
}

// HasConnectionSettings checks if there are any connection settings configured.
func (ci *ConnectionInfo) HasConnectionSettings() bool {
	return ci.opamp.DestinationEndpoint != "" ||
		ci.ownMetrics.DestinationEndpoint != "" ||
		ci.ownLogs.DestinationEndpoint != "" ||
		ci.ownTraces.DestinationEndpoint != "" ||
		len(ci.otherConnections) > 0
}

func (ci *ConnectionInfo) updateHash() error {
	var buf bytes.Buffer

	encoder := json.NewEncoder(&buf)

	err := encoder.Encode(map[string]any{
		"opamp":            ci.opamp,
		"ownMetrics":       ci.ownMetrics,
		"ownLogs":          ci.ownLogs,
		"ownTraces":        ci.ownTraces,
		"otherConnections": ci.otherConnections,
	})
	if err != nil {
		return fmt.Errorf("failed to encode connection info for hash: %w", err)
	}

	ci.Hash, err = vo.NewHash(buf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to compute hash for connection info: %w", err)
	}

	return nil
}

// OpAMPConnectionSettings represents OpAMP connection settings.
type OpAMPConnectionSettings struct {
	DestinationEndpoint string
	Headers             map[string][]string
	Certificate         TelemetryTLSCertificate
}

// TelemetryConnectionSettings represents telemetry connection settings.
type TelemetryConnectionSettings struct {
	DestinationEndpoint string
	Headers             map[string][]string
	Certificate         TelemetryTLSCertificate
}

// OtherConnectionSettings represents other connection settings.
type OtherConnectionSettings struct {
	DestinationEndpoint string
	Headers             map[string][]string
	Certificate         TelemetryTLSCertificate
}

// TelemetryTLSCertificate represents TLS certificate for telemetry.
type TelemetryTLSCertificate struct {
	Cert       []byte
	PrivateKey []byte
	CaCert     []byte
}

// AgentRestartInfo is a domain model to control opamp agent restart information.
type AgentRestartInfo struct {
	// RequiredRestartedAt is the time when the agent is required to be
	// restarted to apply a command that requires a restart.
	RequiredRestartedAt time.Time
}

// AgentComponentHealth is a domain model to control opamp agent component health.
type AgentComponentHealth struct {
	// Set to true if the Agent is up and healthy.
	Healthy bool

	// Timestamp since the Agent is up
	StartTime time.Time

	// Human-readable error message if the Agent is in erroneous state. SHOULD be set when healthy==false.
	LastError string

	// Component status represented as a string.
	// The status values are defined by agent-specific semantics and not at the protocol level.
	Status string

	// The time when the component status was observed.
	StatusTime time.Time

	// A map to store more granular, sub-component health.
	// It can nest as deeply as needed to describe the underlying system.
	ComponentHealthMap map[string]AgentComponentHealth
}

// AgentEffectiveConfig is the effective configuration of the agent.
type AgentEffectiveConfig struct {
	ConfigMap AgentConfigMap
}

// AgentConfigMap is a map of configuration files.
type AgentConfigMap struct {
	// The config_map field of the AgentConfigSet message is a map of configuration files, where keys are file names.
	ConfigMap map[string]AgentConfigFile
}

// AgentConfigFile is a configuration file.
type AgentConfigFile struct {
	// The body field contains the raw bytes of the configuration file.
	// The content, format and encoding of the raw bytes is Agent type-specific and is outside the concerns of OpAMP
	// protocol.
	Body []byte

	// content_type is an optional field. It is a MIME Content-Type that describes what's contained in the body field,
	// for example "text/yaml". The content_type reported in the Effective Configuration in the Agent's status report may
	// be used for example by the Server to visualize the reported configuration nicely in a UI.
	ContentType string
}

// AgentRemoteConfigStatus is the status of the remote configuration.
type AgentRemoteConfigStatus struct {
	LastRemoteConfigHash []byte
	Status               RemoteConfigStatus
	ErrorMessage         string
	LastUpdatedAt        time.Time
}

// AgentConnectionSettingsStatus is the status of the connection settings.
type AgentConnectionSettingsStatus struct {
	LastConnectionSettingsHash []byte
	Status                     ConnectionSettingsStatus
	ErrorMessage               string
}

// ConnectionSettingsStatus represents the status of connection settings.
type ConnectionSettingsStatus int32

const (
	// ConnectionSettingsStatusUnset means status is not set.
	ConnectionSettingsStatusUnset ConnectionSettingsStatus = 0
	// ConnectionSettingsStatusApplied means connection settings have been applied.
	ConnectionSettingsStatusApplied ConnectionSettingsStatus = 1
	// ConnectionSettingsStatusApplying means connection settings are being applied.
	ConnectionSettingsStatusApplying ConnectionSettingsStatus = 2
	// ConnectionSettingsStatusFailed means applying connection settings failed.
	ConnectionSettingsStatusFailed ConnectionSettingsStatus = 3
)

// AgentPackageStatuses is a map of package statuses.
type AgentPackageStatuses struct {
	Packages                     map[string]AgentPackageStatusEntry
	ServerProvidedAllPackgesHash []byte
	ErrorMessage                 string
}

// AgentPackageStatusEntry is the status of a package.
type AgentPackageStatusEntry struct {
	Name                 string
	AgentHasVersion      string
	AgentHasHash         []byte
	ServerOfferedVersion string
	Status               AgentPackageStatusEnum
	ErrorMessage         string
}

// AgentPackageStatusEnum is an enum that represents the status of a package.
type AgentPackageStatusEnum int32

// AgentPackageStatusEnum values
// The AgentPackageStatusEnum enum is defined in the opamp protocol.
const (
	AgentPackageStatusEnumInstalled      = 0
	AgentPackageStatusEnumInstallPending = 1
	AgentPackageStatusEnumInstalling     = 2
	AgentPackageStatusEnumInstallFailed  = 3
	AgentPackageStatusEnumDownloading    = 4
)

// AgentCustomCapabilities is a list of custom capabilities that the Agent supports.
type AgentCustomCapabilities struct {
	Capabilities []string
}

// AgentAvailableComponents is a map of available components.
type AgentAvailableComponents struct {
	Components map[string]ComponentDetails
	Hash       []byte
}

// ComponentDetails is a details of a component.
type ComponentDetails struct {
	Metadata        map[string]string
	SubComponentMap map[string]ComponentDetails
}

// ReportDescription is a method to report the description of the agent.
func (a *Agent) ReportDescription(desc *agent.Description) error {
	if desc == nil {
		return nil // No description to report
	}

	a.Metadata.Description = *desc

	return nil
}

// ReportComponentHealth is a method to report the component health of the agent.
func (a *Agent) ReportComponentHealth(health *AgentComponentHealth) error {
	if health == nil {
		return nil // No health to report
	}

	a.Status.ComponentHealth = *health

	return nil
}

// ReportCapabilities is a method to report the capabilities of the agent.
func (a *Agent) ReportCapabilities(capabilities *agent.Capabilities) error {
	if capabilities == nil {
		return nil // No capabilities to report
	}

	a.Metadata.Capabilities = *capabilities

	return nil
}

// ReportEffectiveConfig is a method to report the effective configuration of the agent.
func (a *Agent) ReportEffectiveConfig(config *AgentEffectiveConfig) error {
	if config == nil {
		return nil // No effective config to report
	}

	a.Status.EffectiveConfig = *config

	return nil
}

// ReportRemoteConfigStatus is a method to report the remote configuration status of the agent.
func (a *Agent) ReportRemoteConfigStatus(status *AgentRemoteConfigStatus) error {
	if status == nil {
		return nil // No remote config status to report
	}

	a.Status.RemoteConfigStatus = *status

	return nil
}

// ReportConnectionSettingsStatus is a method to report the connection settings status of the agent.
func (a *Agent) ReportConnectionSettingsStatus(status *AgentConnectionSettingsStatus) error {
	if status == nil {
		return nil // No connection settings status to report
	}

	a.Status.ConnectionSettingsStatus = *status

	return nil
}

// ReportPackageStatuses is a method to report the package statuses of the agent.
func (a *Agent) ReportPackageStatuses(status *AgentPackageStatuses) error {
	if status == nil {
		return nil // No package statuses to report
	}

	a.Status.PackageStatuses = *status

	return nil
}

// ReportCustomCapabilities is a method to report the custom capabilities of the agent.
func (a *Agent) ReportCustomCapabilities(capabilities *AgentCustomCapabilities) error {
	if capabilities == nil {
		return nil // No custom capabilities to report
	}

	a.Metadata.CustomCapabilities = *capabilities

	return nil
}

// ReportAvailableComponents is a method to report the available components of the agent.
func (a *Agent) ReportAvailableComponents(availableComponents *AgentAvailableComponents) error {
	if availableComponents == nil {
		return nil // No available components to report
	}

	a.Status.AvailableComponents = *availableComponents

	return nil
}

// RecordLastReported updates the last communicated time and server of the agent.
func (a *Agent) RecordLastReported(by *Server, lastReportedAt time.Time, sequenceNum uint64) {
	if by != nil {
		a.Status.LastReportedTo = by.ID
	}

	a.Status.SequenceNum = sequenceNum
	a.Status.LastReportedAt = lastReportedAt
}

// RemoteConfigStatus is generated from agentToServer of OpAMP.
type RemoteConfigStatus int32

// RemoteConfigStatus constants.
const (
	// RemoteConfigStatusUnset means status is not set.
	RemoteConfigStatusUnset RemoteConfigStatus = 0
	// RemoteConfigStatusApplied means remote config has been applied.
	RemoteConfigStatusApplied RemoteConfigStatus = 1
	// RemoteConfigStatusApplying means remote config is being applied.
	RemoteConfigStatusApplying RemoteConfigStatus = 2
	// RemoteConfigStatusFailed means applying remote config failed.
	RemoteConfigStatusFailed RemoteConfigStatus = 3
)

// String returns the string representation of the status.
func (s RemoteConfigStatus) String() string {
	switch s {
	case RemoteConfigStatusUnset:
		return "UNSET"
	case RemoteConfigStatusApplied:
		return "APPLIED"
	case RemoteConfigStatusApplying:
		return "APPLYING"
	case RemoteConfigStatusFailed:
		return "FAILED"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", int32(s))
	}
}

// UpdateLastCommunicationInfo updates the last communication info of the agent.
func (a *Agent) UpdateLastCommunicationInfo(now time.Time, connection *Connection) {
	a.Status.Connected = true

	a.Status.LastReportedAt = now
	if connection != nil {
		a.Status.ConnectionType = connection.Type
	} else {
		a.Status.ConnectionType = ConnectionTypeUnknown
	}
}

// IsRemoteConfigSupported checks if the agent supports remote configuration.
func (a *Agent) IsRemoteConfigSupported() bool {
	return a.Metadata.Capabilities.Has(agent.AgentCapabilityAcceptsRemoteConfig)
}

// HasRemoteConfig checks if the agent has remote configuration to apply.
func (a *Agent) HasRemoteConfig() bool {
	return a.IsRemoteConfigSupported() &&
		len(a.Spec.RemoteConfig.RemoteConfigNames) > 0
}

// SetCondition sets or updates a condition in the agent's status.
func (a *Agent) SetCondition(
	conditionType AgentConditionType,
	status AgentConditionStatus,
	triggeredBy, message string,
) {
	now := time.Now()

	// Check if condition already exists
	for idx, condition := range a.Status.Conditions {
		if condition.Type == conditionType {
			// Update existing condition only if status changed
			if condition.Status != status {
				a.Status.Conditions[idx].Status = status
				a.Status.Conditions[idx].LastTransitionTime = now
				a.Status.Conditions[idx].Reason = triggeredBy
				a.Status.Conditions[idx].Message = message
			}

			return
		}
	}

	// Add new condition
	a.Status.Conditions = append(a.Status.Conditions, AgentCondition{
		Type:               conditionType,
		LastTransitionTime: now,
		Status:             status,
		Reason:             triggeredBy,
		Message:            message,
	})
}

// GetCondition returns the condition of the specified type.
func (a *Agent) GetCondition(conditionType AgentConditionType) *AgentCondition {
	for _, condition := range a.Status.Conditions {
		if condition.Type == conditionType {
			return &condition
		}
	}

	return nil
}

// IsConditionTrue checks if the specified condition type is true.
func (a *Agent) IsConditionTrue(conditionType AgentConditionType) bool {
	condition := a.GetCondition(conditionType)

	return condition != nil && condition.Status == AgentConditionStatusTrue
}

// MarkConnected marks the agent as connected and updates the connection condition.
func (a *Agent) MarkConnected(triggeredBy string) {
	a.Status.Connected = true
	a.Status.LastReportedAt = time.Now()
	a.SetCondition(AgentConditionTypeConnected, AgentConditionStatusTrue, triggeredBy, "Agent connected")
}

// MarkDisconnected marks the agent as disconnected and updates the connection condition.
func (a *Agent) MarkDisconnected(triggeredBy string) {
	a.Status.Connected = false
	a.SetCondition(AgentConditionTypeConnected, AgentConditionStatusFalse, triggeredBy, "Agent disconnected")
}

// MarkHealthy marks the agent as healthy.
func (a *Agent) MarkHealthy(triggeredBy string) {
	a.SetCondition(AgentConditionTypeHealthy, AgentConditionStatusTrue, triggeredBy, "Agent is healthy")
}

// MarkUnhealthy marks the agent as unhealthy.
func (a *Agent) MarkUnhealthy(triggeredBy, reason string) {
	message := "Agent is unhealthy"
	if reason != "" {
		message = "Agent is unhealthy: " + reason
	}

	a.SetCondition(AgentConditionTypeHealthy, AgentConditionStatusFalse, triggeredBy, message)
}

// MarkConfigured marks the agent as configured.
func (a *Agent) MarkConfigured(triggeredBy string) {
	a.SetCondition(AgentConditionTypeConfigured, AgentConditionStatusTrue, triggeredBy, "Agent configuration applied")
}

// NewInstanceUID returns the new instance UID to inform the agent.
func (a *Agent) NewInstanceUID() []byte {
	if a.Spec.NewInstanceUID == uuid.Nil {
		return nil
	}

	return a.Spec.NewInstanceUID[:]
}

// HasNewInstanceUID checks if there is a new instance UID to inform the agent.
func (a *Agent) HasNewInstanceUID() bool {
	return a.Spec.NewInstanceUID != uuid.Nil
}

// HasNewPackages checks if there are new packages available for the agent.
func (a *Agent) HasNewPackages() bool {
	return a.Metadata.Capabilities.HasAcceptsPackages() &&
		len(a.Spec.PackagesAvailable.Packages) > 0
}
