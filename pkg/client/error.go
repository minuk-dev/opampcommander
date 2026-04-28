package client

import (
	"errors"
	"fmt"
	"strings"
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
	StatusCode   int
	ErrorMessage string
}

// Error implements the error interface for ResponseError.
func (e *ResponseError) Error() string {
	msg := e.ErrorMessage
	if isHTMLBody(msg) {
		msg = "unexpected HTML response (server may not be an OpAMP Commander instance)"
	}

	return fmt.Sprintf("response error: status code %d, message: %s", e.StatusCode, msg)
}

func isHTMLBody(s string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(s))

	return strings.HasPrefix(trimmed, "<html") || strings.HasPrefix(trimmed, "<!doctype html")
}
