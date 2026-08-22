package agentmodel

import (
	"time"

	"github.com/google/uuid"
)

// AgentLiveness is the high-churn, low-durability slice of an agent's state.
//
// Every OpAMP heartbeat refreshes these fields, and they are cheap to recompute:
// losing them costs at most one heartbeat interval of accuracy, because the next
// message from the agent restores them. Everything else about an agent
// (description, spec, remote config, effective config, package statuses) is
// durable and stays on the persistence path.
//
// Separating them lets the server absorb heartbeat traffic in a fast tier and
// write through to the durable store on a much slower cadence.
type AgentLiveness struct {
	// InstanceUID identifies the agent this record belongs to.
	InstanceUID uuid.UUID
	// Connected mirrors [AgentStatus.Connected].
	Connected bool
	// ConnectionType mirrors [AgentStatus.ConnectionType].
	ConnectionType ConnectionType
	// SequenceNum mirrors [AgentStatus.SequenceNum].
	SequenceNum uint64
	// LastReportedAt mirrors [AgentStatus.LastReportedAt].
	LastReportedAt time.Time
	// LastReportedTo mirrors [AgentStatus.LastReportedTo]: the ID of the server
	// the agent last reported to.
	LastReportedTo string
	// DurableReportedAt is the LastReportedAt value the durable store currently
	// holds — what was written, not when it was written.
	//
	// That distinction is the whole point. Staleness is measured from the observation
	// the store holds, so recording the write time instead would understate how far
	// behind the document is by however old the observation was when it was flushed,
	// and the write-behind cadence would silently overshoot the staleness window.
	DurableReportedAt time.Time
}

// NewAgentLivenessFromAgent snapshots the liveness fields of an agent.
// DurableReportedAt is left zero; only a write-through moves it.
func NewAgentLivenessFromAgent(agent *Agent) *AgentLiveness {
	if agent == nil {
		return nil
	}

	return &AgentLiveness{
		InstanceUID:       agent.Metadata.InstanceUID,
		Connected:         agent.Status.Connected,
		ConnectionType:    agent.Status.ConnectionType,
		SequenceNum:       agent.Status.SequenceNum,
		LastReportedAt:    agent.Status.LastReportedAt,
		LastReportedTo:    agent.Status.LastReportedTo,
		DurableReportedAt: time.Time{},
	}
}

// Clone returns a deep copy so stores never share a mutable reference with callers.
func (l *AgentLiveness) Clone() *AgentLiveness {
	if l == nil {
		return nil
	}

	cloned := *l

	return &cloned
}

// IsFresherThan reports whether this record observed the agent more recently than
// the given agent document did. A liveness record that is not fresher carries no
// information the document does not already have.
func (l *AgentLiveness) IsFresherThan(agent *Agent) bool {
	if l == nil || agent == nil {
		return false
	}

	return l.LastReportedAt.After(agent.Status.LastReportedAt)
}

// ApplyTo overlays the liveness record onto a persisted agent document, so reads
// reflect the fast tier rather than the last write-through.
//
// The overlay is skipped when the document is already at least as fresh — the
// durable store wins ties. Without that guard a liveness record left behind by a
// crashed node could drag a live agent's state backwards.
func (l *AgentLiveness) ApplyTo(agent *Agent) {
	if !l.IsFresherThan(agent) {
		return
	}

	agent.Status.Connected = l.Connected
	agent.Status.ConnectionType = l.ConnectionType
	agent.Status.SequenceNum = l.SequenceNum
	agent.Status.LastReportedAt = l.LastReportedAt

	if l.LastReportedTo != "" {
		agent.Status.LastReportedTo = l.LastReportedTo
	}
}

// HasUnwrittenObservation reports whether the record holds an observation that has
// not reached the durable store yet.
func (l *AgentLiveness) HasUnwrittenObservation() bool {
	if l == nil || l.LastReportedAt.IsZero() {
		return false
	}

	return l.LastReportedAt.After(l.DurableReportedAt)
}

// IsPendingWriteThroughSince reports whether the record holds an observation that
// has not reached the durable store yet AND was last written through before the
// given cutoff.
//
// The cutoff is how much staleness the caller tolerates in the durable store: pass
// now minus that tolerance and the result is the set of agents whose stored document
// has fallen further behind than it should.
//
// A record that has never been written through is always pending.
func (l *AgentLiveness) IsPendingWriteThroughSince(cutoff time.Time) bool {
	if l == nil || l.LastReportedAt.IsZero() {
		return false
	}

	if !l.HasUnwrittenObservation() {
		return false
	}

	// Inclusive at the boundary: a record exactly at the cutoff is already as stale
	// as the caller tolerates, and skipping it would cost a whole extra cycle.
	return l.DurableReportedAt.IsZero() || !l.DurableReportedAt.After(cutoff)
}

// NeedsPersist reports whether the agent should be written through to the durable
// store, given the minimum interval between write-throughs. A record that has
// never been persisted always needs one.
func (l *AgentLiveness) NeedsPersist(now time.Time, throttle time.Duration) bool {
	if l == nil || l.DurableReportedAt.IsZero() {
		return true
	}

	return now.Sub(l.DurableReportedAt) >= throttle
}

// IsExpiredAt reports whether the record has gone untouched for longer than ttl
// and may be dropped. Used by stores that evict on their own rather than relying
// on a backing store's TTL.
func (l *AgentLiveness) IsExpiredAt(now time.Time, ttl time.Duration) bool {
	if l == nil {
		return true
	}

	anchor := l.LastReportedAt
	if l.DurableReportedAt.After(anchor) {
		anchor = l.DurableReportedAt
	}

	if anchor.IsZero() {
		return true
	}

	return now.Sub(anchor) >= ttl
}
