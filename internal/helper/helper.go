// Package helper provides utility functions for graceful shutdown management.
package helper

import (
	"context"
	"errors"
)

// Shutdowner is an interface for components that need to be shut down gracefully.
type Shutdowner interface {
	Shutdown(ctx context.Context) error
}

// ShutdownListener manages multiple shutdowner instances and coordinates their shutdown.
type ShutdownListener struct {
	Shutdowners []Shutdowner
}

// NewShutdownListener creates a new ShutdownListener instance.
func NewShutdownListener() *ShutdownListener {
	return &ShutdownListener{
		Shutdowners: nil,
	}
}

// Register adds a shutdowner to the list of components to be shut down.
func (s *ShutdownListener) Register(shutdowner Shutdowner) {
	s.Shutdowners = append(s.Shutdowners, shutdowner)
}

// RegisterFunc adds a shutdown function to the list of components to be shut down.
func (s *ShutdownListener) RegisterFunc(shutdownFunc func(ctx context.Context) error) {
	s.Shutdowners = append(s.Shutdowners, shutdownFn(shutdownFunc))
}

type shutdownFn func(ctx context.Context) error

func (f shutdownFn) Shutdown(ctx context.Context) error {
	return f(ctx)
}

// Shutdown calls shutdown on all registered shutdowners and returns any errors that occurred.
func (s *ShutdownListener) Shutdown(ctx context.Context) error {
	var errs []error

	for _, shutdowner := range s.Shutdowners {
		err := shutdowner.Shutdown(ctx)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
