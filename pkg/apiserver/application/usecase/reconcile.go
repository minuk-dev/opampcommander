package usecase

import "context"

// ReconcileManageUsecase re-enforces a domain object's invariants on demand. It dispatches a
// (kind, namespace, name) request to the reconciler registered for that kind, re-running the
// side effects that normally fire when the resource is created or updated. New reconcilable
// kinds become available here without changing this interface.
type ReconcileManageUsecase interface {
	// Reconcile re-enforces the named resource's invariants. For namespace-scoped kinds name
	// is the resource name; for the "agent" kind it is the instance UID. An unknown kind or a
	// missing resource surfaces as an error mapped to 4xx by the transport.
	Reconcile(ctx context.Context, kind string, namespace string, name string) error
	// ReconcileKinds returns the reconcilable kinds, for discovery and CLI completion.
	ReconcileKinds(ctx context.Context) []string
}
