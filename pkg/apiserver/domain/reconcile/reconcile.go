// Package reconcile provides a generic, registry-based mechanism for re-enforcing a
// domain object's invariants on demand. Each reconcilable resource kind registers a
// [Reconciler]; the [Service] dispatches a (kind, namespace, name) request to the
// matching one. New reconcilable objects plug in by implementing [Reconciler] and being
// added to the registry — no new transport/CLI code is required.
package reconcile

import (
	"context"
	"errors"
	"fmt"
	"slices"
)

// ErrUnknownKind is returned when a reconcile is requested for a kind that has no
// registered [Reconciler].
var ErrUnknownKind = errors.New("unknown reconcile kind")

// ErrDuplicateKind is returned when two reconcilers register the same kind.
var ErrDuplicateKind = errors.New("duplicate reconcile kind")

// Reconciler re-enforces the domain invariants for a single resource kind, re-running the
// side effects that normally fire when the resource is created or updated. Implementations
// are thin adapters over the resource's domain use case.
type Reconciler interface {
	// Kind returns the resource kind this reconciler handles (e.g. "agentremoteconfig").
	// It must be unique across the registry and stable, since it is the public selector
	// used by the API and CLI.
	Kind() string
	// Reconcile re-enforces the named resource's invariants. For namespace-scoped kinds
	// name is the resource name; for the agent kind it is the instance UID. A missing
	// resource should surface as port.ErrResourceNotExist so the transport maps it to 404.
	Reconcile(ctx context.Context, namespace, name string) error
}

// Service is the reconcile registry and dispatcher. It indexes the provided reconcilers by
// kind and routes each request to the matching one.
type Service struct {
	byKind map[string]Reconciler
}

// NewService indexes the given reconcilers by kind. It returns an error if two reconcilers
// claim the same kind, since that would make dispatch ambiguous.
func NewService(reconcilers []Reconciler) (*Service, error) {
	byKind := make(map[string]Reconciler, len(reconcilers))

	for _, reconciler := range reconcilers {
		kind := reconciler.Kind()
		if _, exists := byKind[kind]; exists {
			return nil, fmt.Errorf("%w: %q", ErrDuplicateKind, kind)
		}

		byKind[kind] = reconciler
	}

	return &Service{byKind: byKind}, nil
}

// Reconcile dispatches to the reconciler registered for kind. It returns ErrUnknownKind
// when no reconciler is registered for the kind.
func (s *Service) Reconcile(ctx context.Context, kind, namespace, name string) error {
	reconciler, ok := s.byKind[kind]
	if !ok {
		return fmt.Errorf("%w: %q", ErrUnknownKind, kind)
	}

	err := reconciler.Reconcile(ctx, namespace, name)
	if err != nil {
		return fmt.Errorf("reconcile %s %q: %w", kind, name, err)
	}

	return nil
}

// Kinds returns the registered reconcile kinds in sorted order.
func (s *Service) Kinds() []string {
	kinds := make([]string, 0, len(s.byKind))
	for kind := range s.byKind {
		kinds = append(kinds, kind)
	}

	slices.Sort(kinds)

	return kinds
}
