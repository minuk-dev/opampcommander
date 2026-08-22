package agentmodel_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
)

const (
	serverA = "server-a"
	serverB = "server-b"
)

func TestNewAgentLivenessFromAgent(t *testing.T) {
	t.Parallel()

	instanceUID := uuid.New()
	now := time.Date(2026, time.August, 20, 10, 0, 0, 0, time.UTC)

	agent := agentmodel.NewAgent(instanceUID)
	agent.Status.Connected = true
	agent.Status.ConnectionType = agentmodel.ConnectionTypeWebSocket
	agent.Status.SequenceNum = 42
	agent.Status.LastReportedAt = now
	agent.Status.LastReportedTo = serverA

	liveness := agentmodel.NewAgentLivenessFromAgent(agent)

	require.NotNil(t, liveness)
	assert.Equal(t, instanceUID, liveness.InstanceUID)
	assert.True(t, liveness.Connected)
	assert.Equal(t, agentmodel.ConnectionTypeWebSocket, liveness.ConnectionType)
	assert.Equal(t, uint64(42), liveness.SequenceNum)
	assert.Equal(t, now, liveness.LastReportedAt)
	assert.Equal(t, serverA, liveness.LastReportedTo)
	assert.True(t, liveness.LastPersistedAt.IsZero())
}

func TestNewAgentLivenessFromAgent_Nil(t *testing.T) {
	t.Parallel()

	assert.Nil(t, agentmodel.NewAgentLivenessFromAgent(nil))
}

func TestAgentLivenessClone(t *testing.T) {
	t.Parallel()

	original := &agentmodel.AgentLiveness{
		InstanceUID:     uuid.New(),
		Connected:       true,
		ConnectionType:  agentmodel.ConnectionTypeWebSocket,
		SequenceNum:     7,
		LastReportedAt:  time.Now(),
		LastReportedTo:  serverA,
		LastPersistedAt: time.Time{},
	}

	cloned := original.Clone()
	require.NotNil(t, cloned)
	assert.NotSame(t, original, cloned)
	assert.Equal(t, original, cloned)

	cloned.SequenceNum = 8
	assert.Equal(t, uint64(7), original.SequenceNum)

	var nilLiveness *agentmodel.AgentLiveness
	assert.Nil(t, nilLiveness.Clone())
}

func TestAgentLivenessApplyTo(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, time.August, 20, 10, 0, 0, 0, time.UTC)

	testCases := []struct {
		name string
		// documentReportedAt is Status.LastReportedAt on the persisted document.
		documentReportedAt time.Time
		// livenessReportedAt is LastReportedAt on the liveness record.
		livenessReportedAt time.Time
		wantApplied        bool
	}{
		{
			name:               "liveness is fresher",
			documentReportedAt: base,
			livenessReportedAt: base.Add(time.Minute),
			wantApplied:        true,
		},
		{
			name:               "document is fresher",
			documentReportedAt: base.Add(time.Minute),
			livenessReportedAt: base,
			wantApplied:        false,
		},
		{
			name:               "same timestamp keeps the document",
			documentReportedAt: base,
			livenessReportedAt: base,
			wantApplied:        false,
		},
		{
			name:               "liveness over a never-reported document",
			documentReportedAt: time.Time{},
			livenessReportedAt: base,
			wantApplied:        true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			agent := agentmodel.NewAgent(uuid.New())
			agent.Status.Connected = false
			agent.Status.SequenceNum = 1
			agent.Status.LastReportedAt = testCase.documentReportedAt
			agent.Status.LastReportedTo = serverA

			liveness := &agentmodel.AgentLiveness{
				InstanceUID:     agent.Metadata.InstanceUID,
				Connected:       true,
				ConnectionType:  agentmodel.ConnectionTypeWebSocket,
				SequenceNum:     99,
				LastReportedAt:  testCase.livenessReportedAt,
				LastReportedTo:  serverB,
				LastPersistedAt: time.Time{},
			}

			liveness.ApplyTo(agent)

			if testCase.wantApplied {
				assert.True(t, agent.Status.Connected)
				assert.Equal(t, uint64(99), agent.Status.SequenceNum)
				assert.Equal(t, testCase.livenessReportedAt, agent.Status.LastReportedAt)
				assert.Equal(t, serverB, agent.Status.LastReportedTo)

				return
			}

			assert.False(t, agent.Status.Connected)
			assert.Equal(t, uint64(1), agent.Status.SequenceNum)
			assert.Equal(t, testCase.documentReportedAt, agent.Status.LastReportedAt)
			assert.Equal(t, serverA, agent.Status.LastReportedTo)
		})
	}
}

func TestAgentLivenessApplyTo_KeepsServerWhenUnknown(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, time.August, 20, 10, 0, 0, 0, time.UTC)

	agent := agentmodel.NewAgent(uuid.New())
	agent.Status.LastReportedAt = base
	agent.Status.LastReportedTo = serverA

	liveness := &agentmodel.AgentLiveness{
		InstanceUID:     agent.Metadata.InstanceUID,
		Connected:       true,
		ConnectionType:  agentmodel.ConnectionTypeHTTP,
		SequenceNum:     2,
		LastReportedAt:  base.Add(time.Second),
		LastReportedTo:  "",
		LastPersistedAt: time.Time{},
	}

	liveness.ApplyTo(agent)

	assert.Equal(t, serverA, agent.Status.LastReportedTo)
	assert.True(t, agent.Status.Connected)
}

func TestAgentLivenessNeedsPersist(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, time.August, 20, 10, 0, 0, 0, time.UTC)
	throttle := 10 * time.Second

	var never *agentmodel.AgentLiveness
	assert.True(t, never.NeedsPersist(base, throttle))

	fresh := &agentmodel.AgentLiveness{
		InstanceUID: uuid.New(), Connected: true, ConnectionType: agentmodel.ConnectionTypeWebSocket,
		SequenceNum: 1, LastReportedAt: base, LastReportedTo: "", LastPersistedAt: base,
	}
	assert.False(t, fresh.NeedsPersist(base.Add(throttle-time.Nanosecond), throttle))
	assert.True(t, fresh.NeedsPersist(base.Add(throttle), throttle))

	fresh.LastPersistedAt = time.Time{}
	assert.True(t, fresh.NeedsPersist(base, throttle))
}

func TestAgentLivenessIsExpiredAt(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, time.August, 20, 10, 0, 0, 0, time.UTC)
	ttl := time.Minute

	var nilLiveness *agentmodel.AgentLiveness
	assert.True(t, nilLiveness.IsExpiredAt(base, ttl))

	zero := &agentmodel.AgentLiveness{
		InstanceUID: uuid.New(), Connected: false, ConnectionType: agentmodel.ConnectionTypeUnknown,
		SequenceNum: 0, LastReportedAt: time.Time{}, LastReportedTo: "", LastPersistedAt: time.Time{},
	}
	assert.True(t, zero.IsExpiredAt(base, ttl))

	touched := &agentmodel.AgentLiveness{
		InstanceUID: uuid.New(), Connected: true, ConnectionType: agentmodel.ConnectionTypeWebSocket,
		SequenceNum: 1, LastReportedAt: base, LastReportedTo: "", LastPersistedAt: time.Time{},
	}
	assert.False(t, touched.IsExpiredAt(base.Add(ttl-time.Nanosecond), ttl))
	assert.True(t, touched.IsExpiredAt(base.Add(ttl), ttl))

	// A write-through with no newer heartbeat still counts as activity.
	persisted := &agentmodel.AgentLiveness{
		InstanceUID: uuid.New(), Connected: true, ConnectionType: agentmodel.ConnectionTypeWebSocket,
		SequenceNum: 1, LastReportedAt: base, LastReportedTo: "", LastPersistedAt: base.Add(ttl),
	}
	assert.False(t, persisted.IsExpiredAt(base.Add(ttl), ttl))
}
