// Package port provides ports which is defined in the hexagonal architecture.
package port

import "errors"

var (
	// ErrResourceNotExist is an error that indicates that the resource does not exist.
	ErrResourceNotExist = errors.New("resource does not exist")
	// ErrMultipleResourceExist is an error that indicates that multiple resources exist.
	ErrMultipleResourceExist = errors.New("multiple resources exist")
	// ErrInvalidArgument indicates the caller supplied an invalid argument; it maps to HTTP 400.
	ErrInvalidArgument = errors.New("invalid argument")
)
