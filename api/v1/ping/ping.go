// Package ping provides the ping controller for the HTTP API.
package ping

// Response is a struct that represents the response for the ping endpoint.
type Response struct {
	// Message is the response message.
	Message string `json:"message"`
} // @name PingResponse
