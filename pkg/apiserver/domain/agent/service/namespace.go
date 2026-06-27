package agentservice

import (
	"context"
	"fmt"
	"time"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var _ agentport.NamespaceUsecase = (*NamespaceService)(nil)

// NamespaceService provides operations for managing namespaces, including the
// lifecycle rules (uniqueness, default-namespace protection) and the cascade
// delete of a namespace's children.
type NamespaceService struct {
	persistence              agentport.NamespacePersistencePort
	agentGroupUsecase        agentport.AgentGroupUsecase
	certificateUsecase       agentport.CertificateUsecase
	agentPackageUsecase      agentport.AgentPackageUsecase
	agentRemoteConfigUsecase agentport.AgentRemoteConfigUsecase
	tx                       agentport.TransactionPort
	clock                    clock.Clock
	// defaultNamespace is the undeletable built-in namespace name. An empty value
	// falls back to agentmodel.DefaultNamespaceName.
	defaultNamespace string
}

// NewNamespaceService creates a new NamespaceService. An empty defaultNamespace
// falls back to agentmodel.DefaultNamespaceName.
func NewNamespaceService(
	persistence agentport.NamespacePersistencePort,
	agentGroupUsecase agentport.AgentGroupUsecase,
	certificateUsecase agentport.CertificateUsecase,
	agentPackageUsecase agentport.AgentPackageUsecase,
	agentRemoteConfigUsecase agentport.AgentRemoteConfigUsecase,
	txPort agentport.TransactionPort,
	defaultNamespace string,
) *NamespaceService {
	if defaultNamespace == "" {
		defaultNamespace = agentmodel.DefaultNamespaceName
	}

	return &NamespaceService{
		persistence:              persistence,
		agentGroupUsecase:        agentGroupUsecase,
		certificateUsecase:       certificateUsecase,
		agentPackageUsecase:      agentPackageUsecase,
		agentRemoteConfigUsecase: agentRemoteConfigUsecase,
		tx:                       txPort,
		clock:                    clock.NewRealClock(),
		defaultNamespace:         defaultNamespace,
	}
}

// SetClock overrides the clock used for lifecycle timestamps. Intended for tests.
func (s *NamespaceService) SetClock(c clock.Clock) {
	s.clock = c
}

// GetNamespace implements [agentport.NamespaceUsecase].
func (s *NamespaceService) GetNamespace(
	ctx context.Context,
	name string,
	options *model.GetOptions,
) (*agentmodel.Namespace, error) {
	namespace, err := s.persistence.GetNamespace(ctx, name, options)
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace: %w", err)
	}

	return namespace, nil
}

// ListNamespaces implements [agentport.NamespaceUsecase].
func (s *NamespaceService) ListNamespaces(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Namespace], error) {
	resp, err := s.persistence.ListNamespaces(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	return resp, nil
}

// SaveNamespace implements [agentport.NamespaceUsecase].
func (s *NamespaceService) SaveNamespace(
	ctx context.Context,
	namespace *agentmodel.Namespace,
) (*agentmodel.Namespace, error) {
	saved, err := s.persistence.PutNamespace(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to save namespace: %w", err)
	}

	return saved, nil
}

// CreateNamespace implements [agentport.NamespaceUsecase].
func (s *NamespaceService) CreateNamespace(
	ctx context.Context,
	namespace *agentmodel.Namespace,
	actor string,
) (*agentmodel.Namespace, error) {
	name := namespace.Metadata.Name

	// A successful read means the namespace already exists. Any read error is
	// treated as "not found" and the create proceeds, matching the prior behavior.
	existing, getErr := s.persistence.GetNamespace(ctx, name, nil)
	if getErr == nil && existing != nil {
		return nil, fmt.Errorf("%w: %s", agentport.ErrNamespaceAlreadyExists, name)
	}

	namespace.MarkAsCreated(s.clock.Now(), actor)

	saved, err := s.persistence.PutNamespace(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create namespace: %w", err)
	}

	return saved, nil
}

// UpdateNamespace implements [agentport.NamespaceUsecase].
func (s *NamespaceService) UpdateNamespace(
	ctx context.Context,
	name string,
	namespace *agentmodel.Namespace,
) (*agentmodel.Namespace, error) {
	existing, err := s.persistence.GetNamespace(ctx, name, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace for update: %w", err)
	}

	existing.ApplyUpdate(namespace)

	saved, err := s.persistence.PutNamespace(ctx, existing)
	if err != nil {
		return nil, fmt.Errorf("failed to update namespace: %w", err)
	}

	return saved, nil
}

// DeleteNamespace implements [agentport.NamespaceUsecase].
//
// The whole cascade runs inside a single transaction so a mid-cascade failure
// rolls every soft-delete back, preventing a zombie namespace with partially
// deleted children.
func (s *NamespaceService) DeleteNamespace(
	ctx context.Context,
	name string,
	actor string,
) error {
	if name == s.defaultNamespace {
		return agentport.ErrDefaultNamespaceUndeletable
	}

	now := s.clock.Now()

	err := s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		return s.cascadeDeleteNamespace(txCtx, name, now, actor)
	})
	if err != nil {
		return fmt.Errorf("delete namespace %q: %w", name, err)
	}

	return nil
}

// cascadeDeleteNamespace performs the actual cascade. The mongo-driver
// transaction runner may retry the callback on transient errors, so each step
// must be re-runnable (soft-delete sets the same fields and is fine to re-apply).
func (s *NamespaceService) cascadeDeleteNamespace(
	ctx context.Context,
	name string,
	now time.Time,
	deletedBy string,
) error {
	err := s.deleteAgentGroupsInNamespace(ctx, name, now, deletedBy)
	if err != nil {
		return err
	}

	err = s.deleteCertificatesInNamespace(ctx, name, now, deletedBy)
	if err != nil {
		return err
	}

	err = s.deleteAgentPackagesInNamespace(ctx, name, now, deletedBy)
	if err != nil {
		return err
	}

	err = s.deleteAgentRemoteConfigsInNamespace(ctx, name, now, deletedBy)
	if err != nil {
		return err
	}

	namespace, err := s.persistence.GetNamespace(ctx, name, nil)
	if err != nil {
		return fmt.Errorf("get namespace for deletion: %w", err)
	}

	namespace.MarkAsDeleted(now, deletedBy)

	_, err = s.persistence.PutNamespace(ctx, namespace)
	if err != nil {
		return fmt.Errorf("delete namespace row: %w", err)
	}

	return nil
}

func (s *NamespaceService) deleteAgentGroupsInNamespace(
	ctx context.Context,
	name string,
	now time.Time,
	deletedBy string,
) error {
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
			ctx, name, agentGroup.Metadata.Name, now, deletedBy,
		)
		if err != nil {
			return fmt.Errorf(
				"cascade delete agent group %s/%s: %w",
				name, agentGroup.Metadata.Name, err,
			)
		}
	}

	return nil
}

func (s *NamespaceService) deleteCertificatesInNamespace(
	ctx context.Context,
	name string,
	now time.Time,
	deletedBy string,
) error {
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
			ctx, name, certificate.Metadata.Name, now, deletedBy,
		)
		if err != nil {
			return fmt.Errorf(
				"cascade delete certificate %s/%s: %w",
				name, certificate.Metadata.Name, err,
			)
		}
	}

	return nil
}

func (s *NamespaceService) deleteAgentPackagesInNamespace(
	ctx context.Context,
	name string,
	now time.Time,
	deletedBy string,
) error {
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
			ctx, name, agentPackage.Metadata.Name, now, deletedBy,
		)
		if err != nil {
			return fmt.Errorf(
				"cascade delete agent package %s/%s: %w",
				name, agentPackage.Metadata.Name, err,
			)
		}
	}

	return nil
}

func (s *NamespaceService) deleteAgentRemoteConfigsInNamespace(
	ctx context.Context,
	name string,
	now time.Time,
	deletedBy string,
) error {
	agentRemoteConfigs, err := s.agentRemoteConfigUsecase.ListAgentRemoteConfigs(
		ctx, &model.ListOptions{
			Limit:          0,
			Continue:       "",
			IncludeDeleted: false,
		},
	)
	if err != nil {
		return fmt.Errorf("list agent remote configs for cascade: %w", err)
	}

	for _, arc := range agentRemoteConfigs.Items {
		if arc.Metadata.Namespace != name {
			continue
		}

		err = s.agentRemoteConfigUsecase.DeleteAgentRemoteConfig(
			ctx, name, arc.Metadata.Name, now, deletedBy,
		)
		if err != nil {
			return fmt.Errorf(
				"cascade delete agent remote config %s/%s: %w",
				name, arc.Metadata.Name, err,
			)
		}
	}

	return nil
}
