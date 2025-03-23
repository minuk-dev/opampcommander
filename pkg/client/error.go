package client

import (
	"errors"
	"fmt"
)

var (
	ErrEmptyResponse      = errors.New("empty response")
	ErrUnexpectedBehavior = errors.New("unexpected behavior")
)

type ResponseError struct {
	StatusCode int
}

func (e *ResponseError) Error() string {
	return fmt.Sprintf("response error: %d", e.StatusCode)
}
