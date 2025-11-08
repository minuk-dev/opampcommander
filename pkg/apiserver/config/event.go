package config

// EventSettings represents the event settings.
type EventSettings struct {
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

// String returns the string representation of the EventProtocolType.
func (e EventProtocolType) String() string {
	return string(e)
}

const (
	// EventProtocolTypeInMemory represents the in-memory event protocol for standalone mode.
	EventProtocolTypeInMemory EventProtocolType = "inmemory"
	// EventProtocolTypeNATS represents the NATS event protocol.
	EventProtocolTypeNATS EventProtocolType = "nats"
	// TODO: Add kafka protocol support.
	//nolint:godox // too far future
)
