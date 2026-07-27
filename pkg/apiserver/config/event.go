package config

// EventSettings represents the event settings.
type EventSettings struct {
	// ProtocolType is the event protocol type.
	ProtocolType EventProtocolType

	// KafkaSettings represents the Kafka configuration.
	KafkaSettings KafkaSettings

	// DirectSettings represents the direct (peer-to-peer) transport configuration.
	DirectSettings DirectSettings
}

// KafkaSettings represents the Kafka event settings.
type KafkaSettings struct {
	// Brokers is the list of Kafka broker addresses.
	Brokers []string
	// Topic is the Kafka topic name for events.
	Topic string
}

// DirectSettings represents the direct transport settings. In this mode a server
// dials the destination peer directly (resolved from the server registry) instead
// of publishing to a broker, so a targeted message reaches exactly one server.
type DirectSettings struct {
	// SubProtocol selects the wire protocol used between peers (http or grpc).
	SubProtocol DirectSubProtocol
	// ListenAddress is the address the receiver binds to (e.g. ":8081").
	ListenAddress string
	// AdvertiseAddress is the address peers should dial to reach this server
	// (e.g. "10.0.0.5:8081" or "$POD_IP:8081"). It is stored in the server
	// registry and used by other servers to route targeted messages here.
	AdvertiseAddress string
}

// DirectSubProtocol represents the wire protocol used by the direct transport.
type DirectSubProtocol string

// String returns the string representation of the DirectSubProtocol.
func (d DirectSubProtocol) String() string {
	return string(d)
}

const (
	// DirectSubProtocolHTTP uses HTTP/JSON between peers.
	DirectSubProtocolHTTP DirectSubProtocol = "http"
	// DirectSubProtocolGRPC uses gRPC between peers.
	DirectSubProtocolGRPC DirectSubProtocol = "grpc"
)

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
	// EventProtocolTypeDirect represents the direct peer-to-peer event protocol,
	// which routes each targeted message to exactly one server without a broker.
	EventProtocolTypeDirect EventProtocolType = "direct"
)
