package opamp

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opamp-go/server/types"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
)

// handleInstanceUIDConflict keeps one live WebSocket owner per UID. A competing
// session may be a reconnect, so ask it to retry without changing its identity.
// Mutable description attributes cannot distinguish reconnects from duplicates.
// mTLS certificate authorization runs before this handler when required.
func (s *Service) handleInstanceUIDConflict(
	ctx context.Context,
	logger *slog.Logger,
	conn types.Connection,
	instanceUID uuid.UUID,
) *protobufs.ServerToAgent {
	if instanceUID == uuid.Nil || !s.hasLiveConnectionConflict(ctx, conn, instanceUID) {
		return nil
	}

	logger.Warn("instanceUID already has a live WebSocket; deferring competing session",
		slog.String("instanceUID", instanceUID.String()))

	return s.createErrorServerToAgent(instanceUID,
		protobufs.ServerErrorResponseType_ServerErrorResponseType_Unavailable,
		"instance UID already has an active WebSocket session; retry after it disconnects")
}

// hasLiveConnectionConflict reports whether another connection (distinct from conn) is
// currently bound to instanceUID.
func (s *Service) hasLiveConnectionConflict(
	ctx context.Context,
	conn types.Connection,
	instanceUID uuid.UUID,
) bool {
	existingConn, err := s.connectionUsecase.GetConnectionByInstanceUID(ctx, instanceUID)
	if err != nil {
		return false
	}

	// An HTTP connection record represents a poll, not a competing live session.
	if existingConn == nil || existingConn.Type != agentmodel.ConnectionTypeWebSocket {
		return false
	}

	if !existingConn.IsAlive(s.clock.Now()) {
		return false
	}

	currentConn, err := s.connectionUsecase.GetConnectionByID(ctx, conn)
	if err != nil || currentConn == nil {
		// The current connection has not been registered yet. The existing bound
		// connection is therefore from a different network connection.
		return true
	}

	return existingConn.UID != currentConn.UID
}
