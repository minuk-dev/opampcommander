package nats

// UnknownMessageTypeError is returned when the message type is unknown.
type UnknownMessageTypeError struct {
	MessageType string
}

// Error implements the error interface.
func (e *UnknownMessageTypeError) Error() string {
	return "unknown message type: " + e.MessageType
}
