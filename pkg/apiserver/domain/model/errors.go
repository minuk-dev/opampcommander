package model

import "errors"

var (
	// ErrResourceNotExist is an error that indicates that the resource does not exist.
	ErrResourceNotExist = errors.New("resource does not exist")
	// ErrMultipleResourceExist is an error that indicates that multiple resources exist.
	ErrMultipleResourceExist = errors.New("multiple resources exist")
	// ErrInvalidArgument indicates the caller supplied an invalid argument; it maps to HTTP 400.
	ErrInvalidArgument = errors.New("invalid argument")
	// ErrResourceAlreadyExist indicates a create was attempted for a resource that
	// already exists; it maps to HTTP 409.
	ErrResourceAlreadyExist = errors.New("resource already exists")
	// ErrConflict indicates an optimistic-concurrency conflict: the resource was
	// modified by another writer since it was loaded, so the write was rejected to
	// avoid clobbering that change. The caller should re-read and retry. It maps to
	// HTTP 409.
	ErrConflict = errors.New("resource version conflict")
)
