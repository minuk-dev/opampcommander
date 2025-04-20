package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

// CommandService is a struct that implements the CommandUsecase interface.
type CommandService struct {
	commandPersistencePort port.CommandPersistencePort
}

// NewCommandService creates a new instance of CommandService.
func NewCommandService(
	commandPersistencePort port.CommandPersistencePort,
) *CommandService {
	return &CommandService{
		commandPersistencePort: commandPersistencePort,
	}
}

// GetCommand retrieves a command by its ID.
func (s *CommandService) GetCommand(ctx context.Context, commandID uuid.UUID) (*model.Command, error) {
	command, err := s.commandPersistencePort.GetCommand(ctx, commandID)
	if err != nil {
		return nil, fmt.Errorf("failed to get command from persistence: %w", err)
	}

	return command, nil
}

// GetCommandByInstanceUID retrieves a command by its instance UID.
func (s *CommandService) GetCommandByInstanceUID(ctx context.Context, instanceUID uuid.UUID) ([]*model.Command, error) {
	command, err := s.commandPersistencePort.GetCommandByInstanceUID(ctx, instanceUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get command by instance UID from persistence: %w", err)
	}

	return []*model.Command{command}, nil
}

// SaveCommand saves the command to the persistence layer.
func (s *CommandService) SaveCommand(ctx context.Context, command *model.Command) error {
	err := s.commandPersistencePort.SaveCommand(ctx, command)
	if err != nil {
		return fmt.Errorf("failed to save command: %w", err)
	}

	return nil
}
