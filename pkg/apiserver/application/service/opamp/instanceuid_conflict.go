package opamp

import (
	"context"
	"errors"
	"log/slog"
	"maps"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opamp-go/server/types"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

const (
	conflictReasonIdentifyingAttrs   = "stored_agent_has_different_identifying_attributes"
	conflictReasonLiveAndIdentifying = "live_connection_and_identifying_attributes_mismatch"
	conflictTriggeredBy              = "system:opamp"
	serviceInstanceIDAttribute       = "service.instance.id"
)

// instanceUIDConflict describes a detected conflict on the incoming instanceUID.
type instanceUIDConflict struct {
	reason        string
	existingAgent *agentmodel.Agent
}

// handleInstanceUIDConflict runs conflict detection and, when a conflict is found, audits
// the event and returns a renewal ServerToAgent response. Returns nil when no conflict.
func (s *Service) handleInstanceUIDConflict(
	ctx context.Context,
	logger *slog.Logger,
	conn types.Connection,
	instanceUID uuid.UUID,
	message *protobufs.AgentToServer,
) *protobufs.ServerToAgent {
	conflict := s.detectInstanceUIDConflict(ctx, conn, instanceUID, message)
	if conflict == nil {
		return nil
	}

	newInstanceUID := s.newInstanceUID(logger)
	s.recordInstanceUIDConflict(ctx, logger, conflict, instanceUID, newInstanceUID)

	return s.createRenewalServerToAgent(instanceUID, newInstanceUID)
}

// detectInstanceUIDConflict checks whether the incoming message claims an instanceUID that
// is already owned by a different agent identity. A conflict requires the stored agent's
// identifying attributes to differ from the incoming message — a different live connection
// alone is not enough, since rapid reconnects routinely overlap with the asynchronous
// cleanup of the previous Connection record. When a live-connection mismatch coincides with
// an identifying-attribute mismatch, the reason is escalated so operators see the stronger
// signal.
func (s *Service) detectInstanceUIDConflict(
	ctx context.Context,
	conn types.Connection,
	instanceUID uuid.UUID,
	message *protobufs.AgentToServer,
) *instanceUIDConflict {
	if instanceUID == uuid.Nil {
		return nil
	}

	existingAgent, attrsConflict := s.hasIdentifyingAttrsConflict(ctx, instanceUID, message)
	if !attrsConflict {
		return nil
	}

	if s.hasLiveConnectionConflict(ctx, conn, instanceUID) {
		return &instanceUIDConflict{
			reason:        conflictReasonLiveAndIdentifying,
			existingAgent: existingAgent,
		}
	}

	return &instanceUIDConflict{
		reason:        conflictReasonIdentifyingAttrs,
		existingAgent: existingAgent,
	}
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

	if existingConn == nil {
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

// hasIdentifyingAttrsConflict reports whether a stored agent record for instanceUID exists
// and whose identifying attributes differ from those reported in the incoming message.
// Returns the stored agent (or nil) alongside the verdict so callers can attach audit info.
func (s *Service) hasIdentifyingAttrsConflict(
	ctx context.Context,
	instanceUID uuid.UUID,
	message *protobufs.AgentToServer,
) (*agentmodel.Agent, bool) {
	existing, err := s.agentUsecase.GetAgent(ctx, instanceUID)
	if err != nil {
		if !errors.Is(err, model.ErrResourceNotExist) {
			s.logger.Warn("failed to load agent for instanceUID conflict check",
				slog.String("instanceUID", instanceUID.String()),
				slog.String("error", err.Error()))
		}

		return nil, false
	}

	desc := message.GetAgentDescription()
	if desc == nil {
		return existing, false
	}

	incoming := toMap(desc.GetIdentifyingAttributes())
	stored := existing.Metadata.Description.IdentifyingAttributes

	if len(incoming) == 0 || len(stored) == 0 {
		return existing, false
	}

	incomingID, incomingHasID := incoming[serviceInstanceIDAttribute]
	storedID, storedHasID := stored[serviceInstanceIDAttribute]

	if incomingHasID && storedHasID {
		return existing, incomingID != storedID
	}

	return existing, !maps.Equal(incoming, stored)
}

// recordInstanceUIDConflict appends an audit condition onto the stored agent (if any) and
// emits a structured log. The audit captures the old/new instanceUID and the reason so
// operators can trace renewals after the fact.
func (s *Service) recordInstanceUIDConflict(
	ctx context.Context,
	logger *slog.Logger,
	conflict *instanceUIDConflict,
	oldInstanceUID, newInstanceUID uuid.UUID,
) {
	logger.Warn("instanceUID conflict detected; redirecting new arrival to a fresh instanceUID",
		slog.String("oldInstanceUID", oldInstanceUID.String()),
		slog.String("newInstanceUID", newInstanceUID.String()),
		slog.String("reason", conflict.reason),
	)

	if conflict.existingAgent == nil {
		return
	}

	conflict.existingAgent.RecordInstanceUIDConflict(
		s.clock.Now(),
		oldInstanceUID, newInstanceUID,
		conflict.reason,
		conflictTriggeredBy,
	)

	err := s.agentUsecase.SaveAgent(ctx, conflict.existingAgent)
	if err != nil {
		logger.Error("failed to persist instanceUID conflict audit on existing agent",
			slog.String("existingAgentInstanceUID", conflict.existingAgent.Metadata.InstanceUID.String()),
			slog.String("error", err.Error()),
		)
	}
}

// createRenewalServerToAgent builds a ServerToAgent response that asks the agent to switch
// to a freshly issued instanceUID. The InstanceUid field echoes the UID the agent currently
// believes it has, per the OpAMP spec.
func (s *Service) createRenewalServerToAgent(
	oldInstanceUID, newInstanceUID uuid.UUID,
) *protobufs.ServerToAgent {
	return &protobufs.ServerToAgent{
		InstanceUid: oldInstanceUID[:],
		AgentIdentification: &protobufs.AgentIdentification{
			NewInstanceUid: newInstanceUID[:],
		},
	}
}

// newInstanceUID generates a fresh UUID v7 to hand to the conflicting agent. Falls back to
// a random UUID v4 if v7 generation fails (clock/entropy issue) so the protocol can still
// progress.
func (s *Service) newInstanceUID(logger *slog.Logger) uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		logger.Warn("uuid.NewV7 failed; falling back to uuid.New",
			slog.String("error", err.Error()))

		return uuid.New()
	}

	return id
}
