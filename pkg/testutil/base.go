package testutil

import (
	"context"
	"testing"
)

type Base struct {
	Name   string
	Ctx    context.Context
	cancel context.CancelFunc
	T      testing.TB

	ContainerMode bool
}

func NewBase(t testing.TB, opts ...Option) *Base {
	ctx, cancel := context.WithCancel(context.Background())

	base := &Base{
		Name:   "base",
		Ctx:    ctx,
		T:      t,
		cancel: cancel,
	}

	t.Cleanup(func() {
		base.Close()
	})

	for _, opt := range opts {
		opt(base)
	}

	return base
}

type Option func(*Base)

func WithContainerMode() Option {
	return func(b *Base) {
		b.ContainerMode = true
	}
}

func (b *Base) Close() {
	b.cancel()
}
