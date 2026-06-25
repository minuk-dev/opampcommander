package client

import (
	"context"
	"fmt"
)

const (
	// ReconcileURL is the path to reconcile a resource of a given kind in a namespace.
	ReconcileURL = "/api/v1/namespaces/{namespace}/reconcile/{kind}/{name}"
	// ReconcileKindsURL is the path to list the reconcilable kinds.
	ReconcileKindsURL = "/api/v1/reconcile/kinds"
)

// ReconcileService re-enforces a domain object's invariants on demand via the generic
// reconcile endpoint.
type ReconcileService struct {
	service *service
}

// NewReconcileService creates a new ReconcileService.
func NewReconcileService(service *service) *ReconcileService {
	return &ReconcileService{service: service}
}

// Reconcile re-runs the side effects that normally fire on create/update for the named
// resource of the given kind. For kind "agent", name is the instance UID.
func (s *ReconcileService) Reconcile(ctx context.Context, kind, namespace, name string) error {
	// Reconcile runs synchronously on the server and can outlast the shared client's 15s
	// timeout in a large namespace. Clone the client and clear the timeout; the context
	// deadline is the only limit.
	res, err := s.service.Resty.Clone().SetTimeout(0).R().
		SetContext(ctx).
		SetPathParam("namespace", namespace).
		SetPathParam("kind", kind).
		SetPathParam("name", name).
		Post(ReconcileURL)
	if err != nil {
		return fmt.Errorf("failed to reconcile %s %q: %w", kind, name, err)
	}

	if res.IsError() {
		return fmt.Errorf("failed to reconcile %s %q: %w", kind, name, &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	return nil
}

// ListKinds returns the reconcilable kinds advertised by the server.
func (s *ReconcileService) ListKinds(ctx context.Context) ([]string, error) {
	var kinds []string

	res, err := s.service.Resty.R().
		SetContext(ctx).
		SetResult(&kinds).
		Get(ReconcileKindsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to list reconcile kinds: %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to list reconcile kinds: %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	return kinds, nil
}
