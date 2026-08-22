//nolint:testpackage // white-box test of unexported persistence-throttle helpers
package opamp

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

func TestIsHeartbeatOnly(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		msg  *protobufs.AgentToServer
		want bool
	}{
		{
			name: "nil message",
			msg:  nil,
			want: true,
		},
		{
			name: "empty message is a heartbeat",
			msg:  &protobufs.AgentToServer{},
			want: true,
		},
		{
			name: "capabilities-only is a heartbeat — bitmask included on every message",
			msg: &protobufs.AgentToServer{
				Capabilities: 0xff,
			},
			want: true,
		},
		{
			name: "description present is not a heartbeat",
			msg: &protobufs.AgentToServer{
				AgentDescription: &protobufs.AgentDescription{},
			},
			want: false,
		},
		{
			name: "health present is not a heartbeat",
			msg: &protobufs.AgentToServer{
				Health: &protobufs.ComponentHealth{},
			},
			want: false,
		},
		{
			name: "effective config present is not a heartbeat",
			msg: &protobufs.AgentToServer{
				EffectiveConfig: &protobufs.EffectiveConfig{},
			},
			want: false,
		},
		{
			name: "remote config status present is not a heartbeat",
			msg: &protobufs.AgentToServer{
				RemoteConfigStatus: &protobufs.RemoteConfigStatus{},
			},
			want: false,
		},
		{
			name: "package statuses present is not a heartbeat",
			msg: &protobufs.AgentToServer{
				PackageStatuses: &protobufs.PackageStatuses{},
			},
			want: false,
		},
		{
			name: "agent disconnect present is not a heartbeat",
			msg: &protobufs.AgentToServer{
				AgentDisconnect: &protobufs.AgentDisconnect{},
			},
			want: false,
		},
		{
			name: "non-zero flags is not a heartbeat",
			msg: &protobufs.AgentToServer{
				Flags: uint64(protobufs.AgentToServerFlags_AgentToServerFlags_RequestInstanceUid),
			},
			want: false,
		},
		{
			name: "connection settings request present is not a heartbeat",
			msg: &protobufs.AgentToServer{
				ConnectionSettingsRequest: &protobufs.ConnectionSettingsRequest{},
			},
			want: false,
		},
		{
			name: "custom message present is not a heartbeat",
			msg: &protobufs.AgentToServer{
				CustomMessage: &protobufs.CustomMessage{},
			},
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, isHeartbeatOnly(tc.msg))
		})
	}
}

// stubLivenessUsecase answers only the liveness call recordLiveness makes; the
// embedded nil interface panics on anything else, which keeps the fake honest
// about what this test actually exercises.
type stubLivenessUsecase struct {
	agentport.AgentUsecase

	due bool
}

func (s *stubLivenessUsecase) TouchAgentLiveness(
	_ context.Context,
	_ *agentmodel.Agent,
	_ time.Time,
) bool {
	return s.due
}

// recordLivenessFixture wires up a Service with a stopped clock and a liveness
// usecase that answers with the given verdict. The real constructor pulls in too
// many dependencies for a unit test focused on this one decision.
func recordLivenessFixture(now time.Time, usecase agentport.AgentUsecase) *Service {
	return &Service{
		clock:        &persistTestClock{now: now},
		logger:       slog.Default(),
		agentUsecase: usecase,
	}
}

func TestRecordLiveness(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.May, 26, 12, 0, 0, 0, time.UTC)
	agent := agentmodel.NewAgent(uuid.New())

	heartbeat := &protobufs.AgentToServer{}
	nonHeartbeat := &protobufs.AgentToServer{
		Health: &protobufs.ComponentHealth{Healthy: true},
	}

	t.Run("non-heartbeat always persists even inside the throttle window", func(t *testing.T) {
		t.Parallel()

		svc := recordLivenessFixture(now, &stubLivenessUsecase{due: false})

		assert.True(t, svc.recordLiveness(t.Context(), agent, nonHeartbeat, now))
	})

	t.Run("heartbeat persists when the fast tier says the agent is due", func(t *testing.T) {
		t.Parallel()

		svc := recordLivenessFixture(now, &stubLivenessUsecase{due: true})

		assert.True(t, svc.recordLiveness(t.Context(), agent, heartbeat, now))
	})

	t.Run("heartbeat is throttled when the fast tier says it is not due", func(t *testing.T) {
		t.Parallel()

		svc := recordLivenessFixture(now, &stubLivenessUsecase{due: false})

		assert.False(t, svc.recordLiveness(t.Context(), agent, heartbeat, now))
	})
}

// persistTestClock is a fixed clock for the persistence-throttle tests.
// We reuse the existing test clock pattern from server_test.go but keep this
// file self-contained.
type persistTestClock struct {
	now time.Time
}

func (c *persistTestClock) Now() time.Time                  { return c.now }
func (c *persistTestClock) Since(t time.Time) time.Duration { return c.now.Sub(t) }
func (c *persistTestClock) After(d time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1)
	ch <- c.now.Add(d)

	return ch
}
func (c *persistTestClock) NewTimer(_ time.Duration) clock.Timer  { return nil }
func (c *persistTestClock) Sleep(_ time.Duration)                 {}
func (c *persistTestClock) Tick(_ time.Duration) <-chan time.Time { return nil }
