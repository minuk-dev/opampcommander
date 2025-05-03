package agent

// Capabilities is a bitmask of capabilities that the Agent supports.
// The Capabilities enum is defined in the opamp protocol.
type Capabilities uint64

// Has checks if the AgentCapabilities has a specific capability.
func (a *Capabilities) Has(capability Capability) bool {
	if a == nil {
		return false
	}

	return *a&Capabilities(capability) == Capabilities(capability)
}

// HasReportsStatus checks if the AgentCapabilities has the ReportsStatus capability.
func (a *Capabilities) HasReportsStatus() bool {
	return a.Has(AgentCapabilityReportsStatus)
}

// Capability is a helper type to represent the capabilities of the Agent.
// It is used to define the capabilities of the Agent in a more readable way.
type Capability uint64

const (
	// AgentCapabilityUnspecified represents that
	// The capabilities field is unspecified.
	AgentCapabilityUnspecified Capability = 0
	// AgentCapabilityReportsStatus represents that
	// The Agent can report status. This bit MUST be set, since all Agents MUST
	// report status.
	AgentCapabilityReportsStatus Capability = 1
	// AgentCapabilityAcceptsRemoteConfig represents that
	// The Agent can accept remote configuration from the Server.
	AgentCapabilityAcceptsRemoteConfig Capability = 2
	// AgentCapabilityReportsEffectiveConfig represents that
	// The Agent will report EffectiveConfig in AgentToServer.
	AgentCapabilityReportsEffectiveConfig Capability = 4
	// AgentCapabilityAcceptsPackages represents that
	// The Agent can accept package offers.
	// Status: [Beta].
	AgentCapabilityAcceptsPackages Capability = 8
	// AgentCapabilityReportsPackageStatuses represents that
	// The Agent can report package status.
	// Status: [Beta].
	AgentCapabilityReportsPackageStatuses Capability = 16
	// AgentCapabilityReportsOwnTraces represents that
	// The Agent can report own trace to the destination specified by
	// the Server via ConnectionSettingsOffers.own_traces field.
	// Status: [Beta].
	AgentCapabilityReportsOwnTraces Capability = 32
	// AgentCapabilityReportsOwnMetrics represents that
	// The Agent can report own metrics to the destination specified by
	// the Server via ConnectionSettingsOffers.own_metrics field.
	// Status: [Beta].
	AgentCapabilityReportsOwnMetrics Capability = 64
	// AgentCapabilityReportsOwnLogs represents that
	// The Agent can report own logs to the destination specified by
	// the Server via ConnectionSettingsOffers.own_logs field.
	// Status: [Beta].
	AgentCapabilityReportsOwnLogs Capability = 128
	// AgentCapabilityAcceptsOpAMPConnectionSettings represents that
	// The can accept connections settings for OpAMP via
	// ConnectionSettingsOffers.opamp field.
	// Status: [Beta].
	AgentCapabilityAcceptsOpAMPConnectionSettings Capability = 256
	// AgentCapabilityAcceptsOtherConnectionSettings represents that
	// The can accept connections settings for other destinations via
	// ConnectionSettingsOffers.other_connections field.
	// Status: [Beta].
	AgentCapabilityAcceptsOtherConnectionSettings Capability = 512
	// AgentCapabilityAcceptsRestartCommand represents that
	// The Agent can accept restart requests.
	// Status: [Beta].
	AgentCapabilityAcceptsRestartCommand Capability = 1024
	// AgentCapabilityReportsHealth represents that
	// The Agent will report Health via AgentToServer.health field.
	AgentCapabilityReportsHealth Capability = 2048
	// AgentCapabilityReportsRemoteConfig represents that
	// The Agent will report RemoteConfig status via AgentToServer.remote_config_status field.
	AgentCapabilityReportsRemoteConfig Capability = 4096
	// AgentCapabilityReportsHeartbeat represents that
	// The Agent can report heartbeats.
	// This is specified by the ServerToAgent.OpAMPConnectionSettings.heartbeat_interval_seconds field.
	// If this capability is true, but the Server does not set a heartbeat_interval_seconds field, the
	// Agent should use its own configured interval, which by default will be 30s. The Server may not
	// know the configured interval and should not make assumptions about it.
	// Status: [Development].
	AgentCapabilityReportsHeartbeat Capability = 8192
	// AgentCapabilityReportsAvailableComponents represents that
	// The agent will report AvailableComponents via the AgentToServer.available_components field.
	// Status: [Development].
	AgentCapabilityReportsAvailableComponents Capability = 16384
)
