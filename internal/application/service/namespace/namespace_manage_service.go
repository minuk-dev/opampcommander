// Package namespace provides the NamespaceManageService for managing namespaces.
package namespace

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/samber/lo"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/internal/application/helper"
	"github.com/minuk-dev/opampcommander/internal/application/port"
	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/internal/domain/agent/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/security"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

// ErrDefaultNamespaceUndeletable is returned when trying to delete the default namespace.
var ErrDefaultNamespaceUndeletable = errors.New(
	"default namespace cannot be deleted",
)

// ErrNamespaceAlreadyExists is returned when a namespace with the same name already exists.
var ErrNamespaceAlreadyExists = errors.New("namespace already exists")

var _ port.NamespaceManageUsecase = (*Service)(nil)

// Service is a service for managing namespaces.
type Service struct {
	namespaceUsecase         agentport.NamespaceUsecase
	agentGroupUsecase        agentport.AgentGroupUsecase
	certificateUsecase       agentport.CertificateUsecase
	agentPackageUsecase      agentport.AgentPackageUsecase
	agentRemoteConfigUsecase agentport.AgentRemoteConfigUsecase
	mapper                   *helper.Mapper
	clock                    clock.Clock
	logger                   *slog.Logger
}

// NewNamespaceService creates a new namespace manage service.
func NewNamespaceService(
	namespaceUsecase agentport.NamespaceUsecase,
	agentGroupUsecase agentport.AgentGroupUsecase,
	certificateUsecase agentport.CertificateUsecase,
	agentPackageUsecase agentport.AgentPackageUsecase,
	agentRemoteConfigUsecase agentport.AgentRemoteConfigUsecase,
	logger *slog.Logger,
) *Service {
	return &Service{
		namespaceUsecase:         namespaceUsecase,
		agentGroupUsecase:        agentGroupUsecase,
		certificateUsecase:       certificateUsecase,
		agentPackageUsecase:      agentPackageUsecase,
		agentRemoteConfigUsecase: agentRemoteConfigUsecase,
		mapper:                   helper.NewMapper(),
		clock:                    clock.NewRealClock(),
		logger:                   logger,
	}
}

// GetNamespace implements [port.NamespaceManageUsecase].
func (s *Service) GetNamespace(
	ctx context.Context,
	name string,
	options *model.GetOptions,
) (*v1.Namespace, error) {
	ns, err := s.namespaceUsecase.GetNamespace(ctx, name, options)
	if err != nil {
		return nil, fmt.Errorf("get namespace: %w", err)
	}

	return s.mapper.MapNamespaceToAPI(ns), nil
}

// ListNamespaces implements [port.NamespaceManageUsecase].
func (s *Service) ListNamespaces(
	ctx context.Context,
	options *model.ListOptions,
) (*v1.ListResponse[v1.Namespace], error) {
	namespaces, err := s.namespaceUsecase.ListNamespaces(ctx, options)
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

// CreateNamespace implements [port.NamespaceManageUsecase].
func (s *Service) CreateNamespace(
	ctx context.Context,
	apiModel *v1.Namespace,
) (*v1.Namespace, error) {
	name := apiModel.Metadata.Name

	existing, getErr := s.namespaceUsecase.GetNamespace(ctx, name, nil)
	if getErr == nil && existing != nil {
		return nil, fmt.Errorf("%w: %s", ErrNamespaceAlreadyExists, name)
	}

	createdBy, err := security.GetUser(ctx)
	if err != nil {
		s.logger.Warn(
			"failed to get user from context",
			slog.String("error", err.Error()),
		)

		createdBy = security.NewAnonymousUser()
	}

	domainModel := s.mapper.MapAPIToNamespace(apiModel)
	domainModel.MarkAsCreated(s.clock.Now(), createdBy.String())

	saved, err := s.namespaceUsecase.SaveNamespace(ctx, domainModel)
	if err != nil {
		return nil, fmt.Errorf("create namespace: %w", err)
	}

	return s.mapper.MapNamespaceToAPI(saved), nil
}

// UpdateNamespace implements [port.NamespaceManageUsecase].
func (s *Service) UpdateNamespace(
	ctx context.Context,
	name string,
	apiModel *v1.Namespace,
) (*v1.Namespace, error) {
	existing, err := s.namespaceUsecase.GetNamespace(ctx, name, nil)
	if err != nil {
		return nil, fmt.Errorf("get namespace for update: %w", err)
	}

	domainModel := s.mapper.MapAPIToNamespace(apiModel)

	// Preserve immutable fields
	domainModel.Metadata.Name = existing.Metadata.Name
	domainModel.Metadata.CreatedAt = existing.Metadata.CreatedAt
	domainModel.Metadata.DeletedAt = existing.Metadata.DeletedAt
	domainModel.Status = existing.Status

	saved, err := s.namespaceUsecase.SaveNamespace(ctx, domainModel)
	if err != nil {
		return nil, fmt.Errorf("update namespace: %w", err)
	}

	return s.mapper.MapNamespaceToAPI(saved), nil
}

// DeleteNamespace implements [port.NamespaceManageUsecase].
// Cascade deletes all agent groups, certificates, and agent packages in the namespace.
//
//nolint:funlen,cyclop // Cascade delete requires multiple steps.
func (s *Service) DeleteNamespace(
	ctx context.Context,
	name string,
) error {
	if name == agentmodel.DefaultNamespaceName {
		return ErrDefaultNamespaceUndeletable
	}

	deletedBy, err := security.GetUser(ctx)
	if err != nil {
		s.logger.Warn(
			"failed to get user from context",
			slog.String("error", err.Error()),
		)

		deletedBy = security.NewAnonymousUser()
	}

	now := s.clock.Now()

	// Cascade: delete all agent groups in this namespace
	agentGroups, err := s.agentGroupUsecase.ListAgentGroups(
		ctx, &model.ListOptions{
			Limit:          0,
			Continue:       "",
			IncludeDeleted: false,
		},
	)
	if err != nil {
		return fmt.Errorf("list agent groups for cascade: %w", err)
	}

	for _, agentGroup := range agentGroups.Items {
		if agentGroup.Metadata.Namespace != name {
			continue
		}

		err = s.agentGroupUsecase.DeleteAgentGroup(
			ctx,
			name,
			agentGroup.Metadata.Name,
			now,
			deletedBy.String(),
		)
		if err != nil {
			return fmt.Errorf(
				"cascade delete agent group %s/%s: %w",
				name, agentGroup.Metadata.Name, err,
			)
		}
	}

	// Cascade: delete all certificates in this namespace
	certificates, err := s.certificateUsecase.ListCertificate(
		ctx, &model.ListOptions{
			Limit:          0,
			Continue:       "",
			IncludeDeleted: false,
		},
	)
	if err != nil {
		return fmt.Errorf("list certificates for cascade: %w", err)
	}

	for _, certificate := range certificates.Items {
		if certificate.Metadata.Namespace != name {
			continue
		}

		_, err = s.certificateUsecase.DeleteCertificate(
			ctx,
			name,
			certificate.Metadata.Name,
			now,
			deletedBy.String(),
		)
		if err != nil {
			return fmt.Errorf(
				"cascade delete certificate %s/%s: %w",
				name, certificate.Metadata.Name, err,
			)
		}
	}

	// Cascade: delete all agent packages in this namespace
	agentPackages, err := s.agentPackageUsecase.ListAgentPackages(
		ctx, &model.ListOptions{
			Limit:          0,
			Continue:       "",
			IncludeDeleted: false,
		},
	)
	if err != nil {
		return fmt.Errorf("list agent packages for cascade: %w", err)
	}

	for _, agentPackage := range agentPackages.Items {
		if agentPackage.Metadata.Namespace != name {
			continue
		}

		err = s.agentPackageUsecase.DeleteAgentPackage(
			ctx,
			name,
			agentPackage.Metadata.Name,
			now,
			deletedBy.String(),
		)
		if err != nil {
			return fmt.Errorf(
				"cascade delete agent package %s/%s: %w",
				name, agentPackage.Metadata.Name, err,
			)
		}
	}

	// Cascade: delete all agent remote configs in this namespace
	agentRemoteConfigs, err := s.agentRemoteConfigUsecase.ListAgentRemoteConfigs(
		ctx, &model.ListOptions{
			Limit:          0,
			Continue:       "",
			IncludeDeleted: false,
		},
	)
	if err != nil {
		return fmt.Errorf(
			"list agent remote configs for cascade: %w", err,
		)
	}

	for _, arc := range agentRemoteConfigs.Items {
		if arc.Metadata.Namespace != name {
			continue
		}

		err = s.agentRemoteConfigUsecase.DeleteAgentRemoteConfig(
			ctx,
			name,
			arc.Metadata.Name,
			now,
			deletedBy.String(),
		)
		if err != nil {
			return fmt.Errorf(
				"cascade delete agent remote config %s/%s: %w",
				name, arc.Metadata.Name, err,
			)
		}
	}

	err = s.namespaceUsecase.DeleteNamespace(
		ctx, name, now, deletedBy.String(),
	)
	if err != nil {
		return fmt.Errorf("delete namespace: %w", err)
	}

	return nil
}
