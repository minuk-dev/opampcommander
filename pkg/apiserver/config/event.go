package config

// EventSettings represents the event settings.
type EventSettings struct {
	// Enabled indicates whether event handling is enabled.
	// When false, the server runs as a standalone instance without event communication.
	// When true, the server participates in event communication with other servers.
	// Default is false.
	Enabled bool

	// ProtocolType is the event protocol type.
	ProtocolType EventProtocolType

	// NATS settings. Used when ProtocolType is EventProtocolTypeNATS.
	NATS NATSSettings
}

// NATSSettings represents the NATS event settings.
type NATSSettings struct {
	Endpoint      string
	SubjectPrefix string // e.g. "prod.opampcommander."
}

// EventProtocolType represents the type of event protocol.
type EventProtocolType string

const (
	// EventProtocolTypeNATS represents the NATS event protocol.
	EventProtocolTypeNATS EventProtocolType = "nats"
	// TODO: Add kafka protocol support.
	//nolint:godox // too far future
)
