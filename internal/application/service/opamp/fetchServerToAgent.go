package opamp

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
)

// FetchServerToAgent fetch a message.
func (s *Service) FetchServerToAgent(
	ctx context.Context,
	instanceUID uuid.UUID,
) (*protobufs.ServerToAgent, error) {
	conn, err := s.connectionUsecase.GetConnection(instanceUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	s.logger.Info("FetchServerToAgent",
		slog.String("instanceUID", instanceUID.String()),
		slog.String("message", "start"),
	)

	serverToAgent, err := s.createServerToAgent(ctx, instanceUID)
	if err != nil {
		return nil, fmt.Errorf("failed to create server to agent message: %w", err)
	}

	s.logger.Info("FetchServerToAgent",
		slog.String("instanceUID", instanceUID.String()),
		slog.String("message", "success"),
	)

	return serverToAgent, nil
}

func (s *Service) createServerToAgent(ctx context.Context, instanceUID uuid.UUID) (*protobufs.ServerToAgent, error) {
	return &protobufs.ServerToAgent{
		InstanceUid: []byte(instanceUID),
	}, nil
}
