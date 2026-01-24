package service

import (
	"context"
	"fmt"
	"time"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ port.AgentPackageUsecase = (*AgentPackageService)(nil)

// AgentPackageService provides operations for managing agent packages.
type AgentPackageService struct {
	persistence port.AgentPackagePersistencePort
}

// NewAgentPackageService creates a new AgentPackageService.
func NewAgentPackageService(persistence port.AgentPackagePersistencePort) *AgentPackageService {
	return &AgentPackageService{
		persistence: persistence,
	}
}

// GetAgentPackage implements [port.AgentPackageUsecase].
func (s *AgentPackageService) GetAgentPackage(ctx context.Context, name string) (*model.AgentPackage, error) {
	agentPackage, err := s.persistence.GetAgentPackage(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent package: %w", err)
	}
	return agentPackage, nil
}

// ListAgentPackages implements [port.AgentPackageUsecase].
func (s *AgentPackageService) ListAgentPackages(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*model.AgentPackage], error) {
	resp, err := s.persistence.ListAgentPackages(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list agent packages: %w", err)
	}
	return resp, nil
}

// SaveAgentPackage implements [port.AgentPackageUsecase].
func (s *AgentPackageService) SaveAgentPackage(
	ctx context.Context,
	agentPackage *model.AgentPackage,
) (*model.AgentPackage, error) {
	saved, err := s.persistence.PutAgentPackage(ctx, agentPackage)
	if err != nil {
		return nil, fmt.Errorf("failed to save agent package: %w", err)
	}
	return saved, nil
}

// DeleteAgentPackage implements [port.AgentPackageUsecase].
func (s *AgentPackageService) DeleteAgentPackage(
	ctx context.Context,
	name string,
	deletedAt time.Time,
	deletedBy string,
) error {
	agentPackage, err := s.persistence.GetAgentPackage(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to get agent package for deletion: %w", err)
	}

	// Mark as deleted by adding a condition
	agentPackage.Status.Conditions = append(agentPackage.Status.Conditions, model.Condition{
		Type:               model.ConditionTypeDeleted,
		Status:             model.ConditionStatusTrue,
		LastTransitionTime: deletedAt,
		Reason:             deletedBy,
		Message:            "Deleted by " + deletedBy,
	})

	_, err = s.persistence.PutAgentPackage(ctx, agentPackage)
	if err != nil {
		return fmt.Errorf("failed to mark agent package as deleted: %w", err)
	}

	return nil
}
