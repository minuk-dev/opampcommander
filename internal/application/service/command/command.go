// Package command provides the command application usecase for the opampcommander.
package command

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	applicationport "github.com/minuk-dev/opampcommander/internal/application/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ applicationport.CommandLookUpUsecase = (*Service)(nil)

// Service is a struct that implements the CommandLookUpUsecase interface.
type Service struct {
	commandUsecase domainport.CommandUsecase
	logger         *slog.Logger
}

// New creates a new instance of the Service struct.
func New(
	commandUsecase domainport.CommandUsecase,
	logger *slog.Logger,
) *Service {
	return &Service{
		commandUsecase: commandUsecase,
		logger:         logger,
	}
}

// GetCommand implements port.CommandLookUpUsecase.
func (s *Service) GetCommand(ctx context.Context, commandID uuid.UUID) (*model.Command, error) {
	command, err := s.commandUsecase.GetCommand(ctx, commandID)
	if err != nil {
		return nil, fmt.Errorf("failed to get command: %w", err)
	}

	return command, nil
}

// GetCommandByInstanceUID implements port.CommandLookUpUsecase.
func (s *Service) GetCommandByInstanceUID(ctx context.Context, instanceUID uuid.UUID) ([]*model.Command, error) {
	commands, err := s.commandUsecase.GetCommandByInstanceUID(ctx, instanceUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get command by instance UID: %w", err)
	}

	return commands, nil
}

// ListCommands implements port.CommandLookUpUsecase.
func (s *Service) ListCommands(ctx context.Context) ([]*model.Command, error) {
	commands, err := s.commandUsecase.ListCommands(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list commands: %w", err)
	}

	return commands, nil
}
