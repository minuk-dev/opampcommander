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

// PersistAgentLiveness writes an observation held by the fast tier through to the
// durable store, touching only the liveness fields.
//
// This is the write-behind path. It is deliberately not SaveAgent: rewriting the
// whole document on this cadence is the cost the fast tier exists to remove, and it
// would bump the resource version on every heartbeat, so routine liveness would keep
// invalidating concurrent API writes.
//
// It returns [model.ErrResourceNotExist] when the agent is gone, so the caller can
// drop the record instead of chasing it forever.
func (s *AgentService) PersistAgentLiveness(ctx context.Context, liveness *agentmodel.AgentLiveness) error {
	if liveness == nil {
		return nil
	}

	err := s.agentPersistencePort.UpdateAgentLiveness(ctx, liveness)
	if err != nil {
		return fmt.Errorf("failed to persist agent liveness: %w", err)
	}

	// The cached document keeps its now-slightly-stale liveness fields on purpose:
	// reads overlay the fast tier anyway, and invalidating every flushed agent would
	// empty the read cache once per flush cycle.
	s.markLivenessPersisted(ctx, liveness.InstanceUID, liveness.LastReportedAt)

	return nil
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

// markLivenessPersisted records the observation the durable store now holds.
//
// It takes the observation timestamp that was written rather than reading the clock:
// staleness is measured from what the store holds, and anchoring on write time would
// understate it by however old the observation was when it was written.
//
// Every durable write anchors, not just heartbeat-driven ones, so an agent whose
// description just changed does not pay for a second write on its next heartbeat.
// Only the anchor is written, so this cannot clobber an observation another server
// recorded in between.
func (s *AgentService) markLivenessPersisted(
	ctx context.Context,
	instanceUID uuid.UUID,
	reportedAt time.Time,
) {
	err := s.agentLivenessPort.MarkPersisted(ctx, instanceUID, reportedAt)
	if err != nil {
		s.logger.Warn("failed to anchor agent liveness write-through",
			slog.String("instanceUID", instanceUID.String()),
			slog.String("error", err.Error()),
		)
	}
}

// mergeLiveness overlays the agent's fast-tier liveness onto the document read from
// the durable store, so a read reflects the agent's current state rather than the
// last write-through.
//
// The overlay is one-directional and only applies when the record is genuinely
// fresher than the document (see [agentmodel.AgentLiveness.ApplyTo]).
func (s *AgentService) mergeLiveness(ctx context.Context, agent *agentmodel.Agent) {
	if agent == nil {
		return
	}

	s.loadLiveness(ctx, agent.Metadata.InstanceUID).ApplyTo(agent)
}

// mergeLivenessInto overlays fast-tier liveness onto a whole page of agents with a
// single batched read.
//
// Note on filtered listings: options such as ConnectedOnly are evaluated by the
// durable store, before this overlay runs. That stays correct only because the
// write-through interval is bounded well below [agentmodel.DefaultConnectionStaleness]
// — the durable document can lag the fast tier, but never by enough to flip the
// filter's verdict. Post-filtering here instead would break cursor pagination, since
// the page size is decided by the store.
func (s *AgentService) mergeLivenessInto(ctx context.Context, agents []*agentmodel.Agent) {
	if len(agents) == 0 {
		return
	}

	instanceUIDs := make([]uuid.UUID, 0, len(agents))
	for _, agent := range agents {
		instanceUIDs = append(instanceUIDs, agent.Metadata.InstanceUID)
	}

	records, err := s.agentLivenessPort.GetMany(ctx, instanceUIDs)
	if err != nil {
		s.logger.Warn("failed to read agent liveness for listing",
			slog.Int("agents", len(agents)),
			slog.String("error", err.Error()),
		)

		return
	}

	for _, agent := range agents {
		records[agent.Metadata.InstanceUID].ApplyTo(agent)
	}
}

// loadLiveness reads a record from the fast tier, returning nil both when no record
// is held and when the read failed — callers cannot act on the difference, and an
// accelerator's failure must never surface.
func (s *AgentService) loadLiveness(ctx context.Context, instanceUID uuid.UUID) *agentmodel.AgentLiveness {
	liveness, err := s.agentLivenessPort.Get(ctx, instanceUID)
	if err != nil {
		s.logger.Warn("failed to read agent liveness",
			slog.String("instanceUID", instanceUID.String()),
			slog.String("error", err.Error()),
		)

		return nil
	}

	return liveness
}
