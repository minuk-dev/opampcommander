package config

// EventSettings represents the event settings.
type EventSettings struct {
	// ProtocolType is the event protocol type.
	ProtocolType EventProtocolType

	// KafkaSettings represents the Kafka configuration.
	KafkaSettings KafkaSettings
}

// KafkaSettings represents the Kafka event settings.
type KafkaSettings struct {
	// Brokers is the list of Kafka broker addresses.
	Brokers []string
	// Topic is the Kafka topic name for events.
	Topic string
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
	// EventProtocolTypeKafka represents the Kafka event protocol for distributed mode.
	EventProtocolTypeKafka EventProtocolType = "kafka"
)
