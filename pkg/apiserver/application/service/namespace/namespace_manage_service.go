// Package namespace provides the NamespaceManageService for managing namespaces.
package namespace

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/samber/lo"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/helper"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/usecase"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/security"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

// Error aliases for the namespace lifecycle rules. The canonical errors live in
// the domain (agentport); these aliases keep existing references working and let
// the HTTP layer match on them.
var (
	// ErrDefaultNamespaceUndeletable is returned when trying to delete the default namespace.
	ErrDefaultNamespaceUndeletable = agentport.ErrDefaultNamespaceUndeletable
	// ErrNamespaceAlreadyExists is returned when a namespace with the same name already exists.
	ErrNamespaceAlreadyExists = agentport.ErrNamespaceAlreadyExists
)

var _ usecase.NamespaceManageUsecase = (*Service)(nil)

// Service is a service for managing namespaces. It maps between the HTTP DTOs and
// the domain, extracts the acting user from the request context, and delegates
// all namespace business rules (uniqueness, default-namespace protection, cascade
// delete) to the domain NamespaceUsecase.
type Service struct {
	namespaceUsecase agentport.NamespaceUsecase
	mapper           *helper.Mapper
	logger           *slog.Logger
}

// NewNamespaceService creates a new namespace manage service.
func NewNamespaceService(
	namespaceUsecase agentport.NamespaceUsecase,
	logger *slog.Logger,
) *Service {
	return &Service{
		namespaceUsecase: namespaceUsecase,
		mapper:           helper.NewMapper(clock.NewRealClock(), 0),
		logger:           logger,
	}
}

// GetNamespace implements [usecase.NamespaceManageUsecase].
func (s *Service) GetNamespace(
	ctx context.Context,
	name string,
	options *port.GetOptions,
) (*v1.Namespace, error) {
	ns, err := s.namespaceUsecase.GetNamespace(ctx, name, options.ToDomain())
	if err != nil {
		return nil, fmt.Errorf("get namespace: %w", err)
	}

	return s.mapper.MapNamespaceToAPI(ns), nil
}

// ListNamespaces implements [usecase.NamespaceManageUsecase].
func (s *Service) ListNamespaces(
	ctx context.Context,
	options *port.ListOptions,
) (*v1.ListResponse[v1.Namespace], error) {
	namespaces, err := s.namespaceUsecase.ListNamespaces(ctx, options.ToDomain())
	if err != nil {
		return nil, fmt.Errorf("list namespaces: %w", err)
	}

	return &v1.ListResponse[v1.Namespace]{
		Kind:       v1.NamespaceKind,
		APIVersion: v1.APIVersion,
		Metadata: v1.ListMeta{
			Continue:           namespaces.Continue,
			RemainingItemCount: namespaces.RemainingItemCount,
		},
		Items: lo.Map(
			namespaces.Items,
			func(ns *agentmodel.Namespace, _ int) v1.Namespace {
				return *s.mapper.MapNamespaceToAPI(ns)
			},
		),
	}, nil
}

// CreateNamespace implements [usecase.NamespaceManageUsecase].
func (s *Service) CreateNamespace(
	ctx context.Context,
	apiModel *v1.Namespace,
) (*v1.Namespace, error) {
	domainModel := s.mapper.MapAPIToNamespace(apiModel)

	created, err := s.namespaceUsecase.CreateNamespace(ctx, domainModel, s.actor(ctx))
	if err != nil {
		return nil, fmt.Errorf("create namespace: %w", err)
	}

	return s.mapper.MapNamespaceToAPI(created), nil
}

// UpdateNamespace implements [usecase.NamespaceManageUsecase].
func (s *Service) UpdateNamespace(
	ctx context.Context,
	name string,
	apiModel *v1.Namespace,
) (*v1.Namespace, error) {
	domainModel := s.mapper.MapAPIToNamespace(apiModel)

	updated, err := s.namespaceUsecase.UpdateNamespace(ctx, name, domainModel)
	if err != nil {
		return nil, fmt.Errorf("update namespace: %w", err)
	}

	return s.mapper.MapNamespaceToAPI(updated), nil
}

// DeleteNamespace implements [usecase.NamespaceManageUsecase]. The cascade delete and
// the default-namespace guard live in the domain NamespaceUsecase.
func (s *Service) DeleteNamespace(
	ctx context.Context,
	name string,
) error {
	err := s.namespaceUsecase.DeleteNamespace(ctx, name, s.actor(ctx))
	if err != nil {
		return fmt.Errorf("delete namespace %q: %w", name, err)
	}

	return nil
}

// actor resolves the acting user from the request context, falling back to an
// anonymous identity (and logging) when none is present.
func (s *Service) actor(ctx context.Context) string {
	user, err := security.GetUser(ctx)
	if err != nil {
		s.logger.Warn("failed to get user from context", slog.String("error", err.Error()))

		user = security.NewAnonymousUser()
	}

	return user.String()
}
