// Package port provides ports which is defined in the hexagonal architecture.
package port

import "errors"

var (
	// ErrResourceNotExist is an error that indicates that the resource does not exist.
	ErrResourceNotExist = errors.New("resource does not exist")
	// ErrMultipleResourceExist is an error that indicates that multiple resources exist.
	ErrMultipleResourceExist = errors.New("multiple resources exist")
)
