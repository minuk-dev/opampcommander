// Package common provides shared adapter infrastructure used by both primary
// (inbound) and secondary (outbound) adapters.
//
// In standalone mode a single in-memory event hub backs both the outbound sender
// (secondary) and the inbound receiver (primary), so the hub instance is shared
// here. Direction-specific transports (the Kafka sender/receiver) live in the
// secondary/primary adapters respectively.
package common

import (
	"go.uber.org/fx"
)

// New creates the common adapter module.
func New() fx.Option {
	return fx.Options(
		fx.Provide(newInMemoryEventHub),
	)
}

// UnsupportedEventProtocolError is returned when an unsupported event protocol type is specified.
type UnsupportedEventProtocolError struct {
	ProtocolType string
}

// Error implements the error interface.
func (e *UnsupportedEventProtocolError) Error() string {
	return "unsupported event protocol type: " + e.ProtocolType
}
