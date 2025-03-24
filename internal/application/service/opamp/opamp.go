package opamp

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	applicationport "github.com/minuk-dev/opampcommander/internal/application/port"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ applicationport.OpAMPUsecase = (*Service)(nil)

type Service struct {
	logger            *slog.Logger
	connectionUsecase domainport.ConnectionUsecase
	agentUsecase      domainport.AgentUsecase
	commandUsecase    domainport.CommandUsecase
}

func New(
	connectionUsecase domainport.ConnectionUsecase,
	agentUsecase domainport.AgentUsecase,
	commandUsecase domainport.CommandUsecase,
	logger *slog.Logger,
) *Service {
	return &Service{
		logger:            logger,
		connectionUsecase: connectionUsecase,
		agentUsecase:      agentUsecase,
		commandUsecase:    commandUsecase,
	}
}

func (s *Service) DisconnectAgent(instanceUID uuid.UUID) error {
	conn, err := s.connectionUsecase.FetchAndDeleteConnection(instanceUID)
	if err != nil && errors.Is(err, domainport.ErrConnectionNotFound) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to delete connection: %w", err)
	}

	if conn != nil {
		return fmt.Errorf("connection is nil: %w", err)
	}

	err = conn.Close()
	if err != nil {
		return fmt.Errorf("failed to close connection: %w", err)
	}

	return nil
}
