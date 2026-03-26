package agentservice

import (
	"context"
	"fmt"
	"time"

	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/internal/domain/agent/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

var _ agentport.AgentPackageUsecase = (*AgentPackageService)(nil)

// AgentPackageService provides operations for managing agent packages.
type AgentPackageService struct {
	persistence agentport.AgentPackagePersistencePort
}

// NewAgentPackageService creates a new AgentPackageService.
func NewAgentPackageService(persistence agentport.AgentPackagePersistencePort) *AgentPackageService {
	return &AgentPackageService{
		persistence: persistence,
	}
}

// GetAgentPackage implements [agentport.AgentPackageUsecase].
func (s *AgentPackageService) GetAgentPackage(ctx context.Context, name string) (*agentmodel.AgentPackage, error) {
	agentPackage, err := s.persistence.GetAgentPackage(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent package: %w", err)
	}

	return agentPackage, nil
}

// ListAgentPackages implements [agentport.AgentPackageUsecase].
func (s *AgentPackageService) ListAgentPackages(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.AgentPackage], error) {
	resp, err := s.persistence.ListAgentPackages(ctx, options)
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

// DeleteAgentPackage implements [agentport.AgentPackageUsecase].
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

	agentPackage.MarkAsDeleted(deletedAt, deletedBy)

	_, err = s.persistence.PutAgentPackage(ctx, agentPackage)
	if err != nil {
		return fmt.Errorf("failed to mark agent package as deleted: %w", err)
	}

	return nil
}
