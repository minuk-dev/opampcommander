package client

import (
	"errors"
	"fmt"
)

var (
	// ErrEmptyResponse is an error that indicates that the response from the server is empty.
	ErrEmptyResponse = errors.New("empty response")
	// ErrUnexpectedBehavior is an error that indicates that the server behavior is not as expected.
	ErrUnexpectedBehavior = errors.New("unexpected behavior")
)

// ResponseError represents an error that occurs when the response from the server is not as expected.
// It contains the status code of the response.
type ResponseError struct {
	StatusCode int
}

// Error implements the error interface for ResponseError.
func (e *ResponseError) Error() string {
	return fmt.Sprintf("response error: %d", e.StatusCode)
}
