package opamp

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
)

// FetchServerToAgent fetch a message.
func (s *Service) FetchServerToAgent(
	ctx context.Context,
	instanceUID uuid.UUID,
) (*protobufs.ServerToAgent, error) {
	s.logger.Info("FetchServerToAgent",
		slog.String("instanceUID", instanceUID.String()),
		slog.String("message", "start"),
	)

	serverToAgent := s.createServerToAgent(ctx, instanceUID)

	s.logger.Info("FetchServerToAgent",
		slog.String("instanceUID", instanceUID.String()),
		slog.String("message", "success"),
	)

	return serverToAgent, nil
}

func (s *Service) createServerToAgent(_ context.Context, instanceUID uuid.UUID) *protobufs.ServerToAgent {
	//exhaustruct:ignore
	return &protobufs.ServerToAgent{
		InstanceUid: instanceUID[:],
	}
}
