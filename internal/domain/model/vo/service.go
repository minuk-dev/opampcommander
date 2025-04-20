package vo

// Service is a value object that represents a service which is defined in opentelemetry's semantic conventions.
type Service struct {
	Name       string
	Namespace  string
	Version    string
	InstanceID string
}
