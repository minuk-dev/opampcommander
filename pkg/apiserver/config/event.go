package config

import (
	"errors"
	"fmt"
	"time"
)

// ErrKafkaSendSettingsInvalid indicates negative Kafka delivery settings.
var ErrKafkaSendSettingsInvalid = errors.New("kafka delivery settings must not be negative")

const (
	defaultKafkaSendTimeout      = 2 * time.Second
	defaultKafkaRetryBackoff     = 100 * time.Millisecond
	defaultKafkaRetryAttempts    = 2
	defaultKafkaFailureThreshold = 2
	defaultKafkaProbeInterval    = 5 * time.Second
)

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
	Brokers []string `mapstructure:"brokers"`
	// Topic is the Kafka topic name for events.
	Topic string `mapstructure:"topic"`
	// SendTimeout bounds all attempts for a single event, including backoff. Default: 2s.
	SendTimeout time.Duration `mapstructure:"sendTimeout"`
	// RetryBackoff is the delay between successive send attempts. Default: 100ms.
	RetryBackoff time.Duration `mapstructure:"retryBackoff"`
	// RetryAttempts includes the initial attempt; 1 disables retries. Default: 2.
	RetryAttempts int `mapstructure:"retryAttempts"`
	// FailureThreshold is the number of failed sends that opens the breaker. Default: 2.
	FailureThreshold int `mapstructure:"failureThreshold"`
	// ProbeInterval is how long the breaker stays open before allowing a probe. Default: 5s.
	ProbeInterval time.Duration `mapstructure:"probeInterval"`
}

// DefaultKafkaSettings returns the default Kafka delivery settings.
func DefaultKafkaSettings() KafkaSettings {
	return KafkaSettings{}.WithDefaults()
}

// WithDefaults replaces zero delivery settings with defaults, preserving explicit overrides.
func (s KafkaSettings) WithDefaults() KafkaSettings {
	if s.SendTimeout == 0 {
		s.SendTimeout = defaultKafkaSendTimeout
	}

	if s.RetryBackoff == 0 {
		s.RetryBackoff = defaultKafkaRetryBackoff
	}

	if s.RetryAttempts == 0 {
		s.RetryAttempts = defaultKafkaRetryAttempts
	}

	if s.FailureThreshold == 0 {
		s.FailureThreshold = defaultKafkaFailureThreshold
	}

	if s.ProbeInterval == 0 {
		s.ProbeInterval = defaultKafkaProbeInterval
	}

	return s
}

// Validate rejects negative settings; zero means use the default.
func (s KafkaSettings) Validate() error {
	if s.SendTimeout < 0 || s.RetryBackoff < 0 || s.RetryAttempts < 0 ||
		s.FailureThreshold < 0 || s.ProbeInterval < 0 {
		return fmt.Errorf("%w: sendTimeout, retryBackoff, retryAttempts, failureThreshold, probeInterval",
			ErrKafkaSendSettingsInvalid)
	}

	return nil
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
	// AuthToken is a pre-shared bearer credential. When set, the receiver requires
	// each incoming delivery to present it and the sender attaches it. When empty,
	// peers are trusted (suitable only for an isolated cluster network).
	AuthToken string
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
