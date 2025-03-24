package admin

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	applicationport "github.com/minuk-dev/opampcommander/internal/application/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ applicationport.AdminUsecase = (*Service)(nil)

type Service struct {
	logger         *slog.Logger
	agentUsecase   domainport.AgentUsecase
	commandUsecase domainport.CommandUsecase
}

func New(
	agentUsecase domainport.AgentUsecase,
	commandUsecase domainport.CommandUsecase,
	logger *slog.Logger,
) *Service {
	return &Service{
		logger:         logger,
		agentUsecase:   agentUsecase,
		commandUsecase: commandUsecase,
	}
}

func (s *Service) ApplyRawConfig(ctx context.Context, targetInstanceUID uuid.UUID, config any) error {
	command := model.NewUpdateAgentConfigCommand(targetInstanceUID, config)

	err := s.commandUsecase.SaveCommand(ctx, command)
	if err != nil {
		return fmt.Errorf("failed to save command: %w", err)
	}

	return nil
}
