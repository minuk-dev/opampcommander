package helper

import (
	"context"
	"errors"
)

type Shutdowner interface {
	Shutdown(ctx context.Context) error
}

type ShutdownListener struct {
	Shutdowners []Shutdowner
}

func NewShutdownListener() *ShutdownListener {
	return &ShutdownListener{
		Shutdowners: nil,
	}
}

func (s *ShutdownListener) Register(shutdowner Shutdowner) {
	s.Shutdowners = append(s.Shutdowners, shutdowner)
}

func (s *ShutdownListener) RegisterFunc(shutdownFunc func(ctx context.Context) error) {
	s.Shutdowners = append(s.Shutdowners, shutdownFn(shutdownFunc))
}

type shutdownFn func(ctx context.Context) error

func (f shutdownFn) Shutdown(ctx context.Context) error {
	return f(ctx)
}

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
