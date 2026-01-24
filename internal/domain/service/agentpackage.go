package service

import (
	"context"
	"time"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ port.AgentPackageUsecase = (*AgentPackageService)(nil)

type AgentPackageService struct {
}

// GetAgentPackage implements [port.AgentPackageUsecase].
func (a *AgentPackageService) GetAgentPackage(ctx context.Context, name string) (*model.AgentPackage, error) {
	panic("unimplemented")
}

// ListAgentPackages implements [port.AgentPackageUsecase].
func (a *AgentPackageService) ListAgentPackages(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.AgentPackage], error) {
	panic("unimplemented")
}

// SaveAgentPackage implements [port.AgentPackageUsecase].
func (a *AgentPackageService) SaveAgentPackage(ctx context.Context, agentPackage *model.AgentPackage) (*model.AgentPackage, error) {
	panic("unimplemented")
}

// DeleteAgentPackage implements [port.AgentPackageUsecase].
func (a *AgentPackageService) DeleteAgentPackage(ctx context.Context, name string, deletedAt time.Time, deletedBy string) error {
	panic("unimplemented")
}
