package agentservice

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
)

// TouchAgentLiveness records the agent's liveness in the fast tier and reports
// whether the agent is due for a write-through to the durable store.
//
// This is the hot path: it runs on every incoming agent message, including the
// heartbeats that carry no state change at all. It costs exactly one call into the
// fast tier — the store preserves the write-through anchor and hands the merged
// record back, so nothing has to be read first.
//
// The fast tier is an accelerator, never a source of truth, so it cannot fail this
// call. A store that cannot answer reports the agent as due, degrading to a durable
// write rather than to a lost heartbeat.
func (s *AgentService) TouchAgentLiveness(
	ctx context.Context,
	agent *agentmodel.Agent,
	observedAt time.Time,
) bool {
	if agent == nil {
		return false
	}

	observation := agentmodel.NewAgentLivenessFromAgent(agent)
	observation.LastReportedAt = observedAt

	stored, err := s.agentLivenessPort.Touch(ctx, observation)
	if err != nil || stored == nil {
		if err != nil {
			s.logger.Warn("failed to record agent liveness",
				slog.String("instanceUID", agent.Metadata.InstanceUID.String()),
				slog.String("error", err.Error()),
			)
		}

		return true
	}

	return stored.NeedsPersist(s.clock.Now(), s.livenessPersistThrottle)
}

// ForgetAgentLiveness drops the agent's liveness record.
//
// Called on a genuine disconnect so the first message after a reconnect is written
// through immediately instead of waiting out the throttle window left behind by the
// previous session.
func (s *AgentService) ForgetAgentLiveness(ctx context.Context, instanceUID uuid.UUID) error {
	err := s.agentLivenessPort.Delete(ctx, instanceUID)
	if err != nil {
		s.logger.Warn("failed to forget agent liveness",
			slog.String("instanceUID", instanceUID.String()),
			slog.String("error", err.Error()),
		)

		return fmt.Errorf("failed to forget agent liveness: %w", err)
	}

	return nil
}

// markLivenessPersisted anchors the write-through throttle at now, after the agent
// has been written to the durable store. Every durable write resets the window,
// not just the heartbeat-driven ones, so an agent whose description just changed
// does not immediately pay for a second write on its next heartbeat.
//
// It writes only the anchor, so it cannot clobber an observation another server
// recorded in between.
func (s *AgentService) markLivenessPersisted(ctx context.Context, instanceUID uuid.UUID) {
	err := s.agentLivenessPort.MarkPersisted(ctx, instanceUID, s.clock.Now())
	if err != nil {
		s.logger.Warn("failed to anchor agent liveness write-through",
			slog.String("instanceUID", instanceUID.String()),
			slog.String("error", err.Error()),
		)
	}
}
