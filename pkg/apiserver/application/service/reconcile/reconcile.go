// Package reconcile provides the application service that exposes the generic domain
// reconcile registry as a management use case.
package reconcile

import (
	"context"
	"errors"
	"fmt"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/usecase"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	domainreconcile "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/reconcile"
)

var _ usecase.ReconcileManageUsecase = (*Service)(nil)

// Service implements usecase.ReconcileManageUsecase by delegating to the domain reconcile
// registry. It owns no logic of its own — each kind's behavior lives in its Reconciler.
type Service struct {
	registry *domainreconcile.Service
}

// New creates a new reconcile application Service.
func New(registry *domainreconcile.Service) *Service {
	return &Service{registry: registry}
}

// Reconcile implements usecase.ReconcileManageUsecase. It translates an unknown-kind error to
// an invalid-argument so the transport maps it to 400 rather than 500; not-found and other
// errors from the reconciler propagate unchanged for their normal status mapping.
func (s *Service) Reconcile(ctx context.Context, kind string, namespace string, name string) error {
	err := s.registry.Reconcile(ctx, kind, namespace, name)
	if err != nil {
		if errors.Is(err, domainreconcile.ErrUnknownKind) {
			return fmt.Errorf("%w: %w", model.ErrInvalidArgument, err)
		}

		return fmt.Errorf("reconcile: %w", err)
	}

	return nil
}

// ReconcileKinds implements usecase.ReconcileManageUsecase.
func (s *Service) ReconcileKinds(_ context.Context) []string {
	return s.registry.Kinds()
}
