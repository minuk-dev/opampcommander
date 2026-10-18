package agentservice

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"time"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var _ agentport.AgentPackageUsecase = (*AgentPackageService)(nil)

// AgentPackageService provides operations for managing agent packages, including
// the creation/update lifecycle rules (stamping and immutable-field preservation).
type AgentPackageService struct {
	persistence agentport.AgentPackagePersistencePort
	clock       clock.Clock
}

// NewAgentPackageService creates a new AgentPackageService.
func NewAgentPackageService(persistence agentport.AgentPackagePersistencePort) *AgentPackageService {
	return &AgentPackageService{
		persistence: persistence,
		clock:       clock.NewRealClock(),
	}
}

// SetClock overrides the clock used for lifecycle timestamps. Intended for tests.
func (s *AgentPackageService) SetClock(c clock.Clock) {
	s.clock = c
}

// GetAgentPackage implements [agentport.AgentPackageUsecase].
func (s *AgentPackageService) GetAgentPackage(
	ctx context.Context,
	namespace string,
	name string,
	options *model.GetOptions,
) (*agentmodel.AgentPackage, error) {
	agentPackage, err := s.persistence.GetAgentPackage(ctx, namespace, name, options)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent package: %w", err)
	}

	return agentPackage, nil
}

// ListAgentPackages implements [agentport.AgentPackageUsecase].
func (s *AgentPackageService) ListAgentPackages(
	ctx context.Context,
	namespace string,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.AgentPackage], error) {
	resp, err := s.persistence.ListAgentPackages(ctx, namespace, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list agent packages: %w", err)
	}

	return resp, nil
}

// SaveAgentPackage implements [agentport.AgentPackageUsecase].
func (s *AgentPackageService) SaveAgentPackage(
	ctx context.Context,
	agentPackage *agentmodel.AgentPackage,
) (*agentmodel.AgentPackage, error) {
	saved, err := s.persistence.PutAgentPackage(ctx, agentPackage)
	if err != nil {
		return nil, fmt.Errorf("failed to save agent package: %w", err)
	}

	return saved, nil
}

// CreateAgentPackage implements [agentport.AgentPackageUsecase].
func (s *AgentPackageService) CreateAgentPackage(
	ctx context.Context,
	agentPackage *agentmodel.AgentPackage,
	actor string,
) (*agentmodel.AgentPackage, error) {
	// Reject creating over an existing package instead of silently upserting it,
	// which would overwrite it and rewind its optimistic-concurrency version.
	_, err := s.persistence.GetAgentPackage(ctx,
		agentPackage.Metadata.Namespace, agentPackage.Metadata.Name,
		&model.GetOptions{IncludeDeleted: true})
	switch {
	case err == nil:
		return nil, fmt.Errorf("%w: agent package %q in namespace %q",
			model.ErrResourceAlreadyExist, agentPackage.Metadata.Name, agentPackage.Metadata.Namespace)
	case !errors.Is(err, model.ErrResourceNotExist):
		return nil, fmt.Errorf("check existing agent package: %w", err)
	}

	agentPackage.Metadata.DeletedAt = nil
	agentPackage.Metadata.ResourceVersion = 0
	agentPackage.Status.Conditions = nil
	agentPackage.MarkAsCreated(s.clock.Now(), actor)

	created, err := s.persistence.PutAgentPackage(ctx, agentPackage)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent package: %w", err)
	}

	return created, nil
}

// UpdateAgentPackage implements [agentport.AgentPackageUsecase].
func (s *AgentPackageService) UpdateAgentPackage(
	ctx context.Context,
	namespace string,
	name string,
	agentPackage *agentmodel.AgentPackage,
) (*agentmodel.AgentPackage, error) {
	err := model.CheckResourceIdentity(namespace,
		name, agentPackage.Metadata.Namespace, agentPackage.Metadata.Name)
	if err != nil {
		return nil, fmt.Errorf("resource precondition: %w", err)
	}

	existing, err := s.persistence.GetAgentPackage(ctx, namespace, name, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent package for update: %w", err)
	}

	if agentPackage.Metadata.ResourceVersion <= 0 {
		return nil, fmt.Errorf("%w: metadata.resourceVersion is required", model.ErrInvalidArgument)
	}

	if existing.Spec.Equal(agentPackage.Spec) &&
		maps.Equal(existing.Metadata.Attributes, agentPackage.Metadata.Attributes) {
		return existing, nil
	}

	err = model.CheckResourceVersion(agentPackage.Metadata.ResourceVersion,
		existing.Metadata.ResourceVersion)
	if err != nil {
		return nil, fmt.Errorf("resource precondition: %w", err)
	}

	existing.ApplyUpdate(agentPackage)

	updated, err := s.persistence.PutAgentPackage(ctx, existing)
	if err != nil {
		return nil, fmt.Errorf("failed to update agent package: %w", err)
	}

	return updated, nil
}

// DeleteAgentPackage implements [agentport.AgentPackageUsecase].
//
//nolint:dupl // Resource deletion shares the revision and tombstone contract.
func (s *AgentPackageService) DeleteAgentPackage(
	ctx context.Context,
	namespace string,
	name string,
	deletedAt time.Time,
	deletedBy string,
	resourceVersion ...int64,
) error {
	agentPackage, err := s.persistence.GetAgentPackage(ctx, namespace, name, &model.GetOptions{IncludeDeleted: true})
	if err != nil {
		return fmt.Errorf("failed to get agent package for deletion: %w", err)
	}

	if len(resourceVersion) > 0 {
		if agentPackage.Metadata.DeletedAt != nil && resourceVersion[0] == agentPackage.Metadata.ResourceVersion-1 {
			return nil
		}

		err = model.CheckResourceVersion(resourceVersion[0], agentPackage.Metadata.ResourceVersion)
		if err != nil {
			return fmt.Errorf("resource precondition: %w", err)
		}
	}

	if agentPackage.Metadata.DeletedAt != nil {
		return nil
	}

	agentPackage.MarkAsDeleted(deletedAt, deletedBy)

	_, err = s.persistence.PutAgentPackage(ctx, agentPackage)
	if err != nil {
		return fmt.Errorf("failed to mark agent package as deleted: %w", err)
	}

	return nil
}
