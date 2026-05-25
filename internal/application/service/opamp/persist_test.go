//nolint:testpackage // white-box test of unexported persistence-throttle helpers
package opamp

import (
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"

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
			msg:  &protobufs.AgentToServer{}, //nolint:exhaustruct // testing zero value
			want: true,
		},
		{
			name: "capabilities-only is a heartbeat — bitmask included on every message",
			msg: &protobufs.AgentToServer{ //nolint:exhaustruct // testing single field
				Capabilities: 0xff,
			},
			want: true,
		},
		{
			name: "description present is not a heartbeat",
			msg: &protobufs.AgentToServer{ //nolint:exhaustruct
				AgentDescription: &protobufs.AgentDescription{}, //nolint:exhaustruct
			},
			want: false,
		},
		{
			name: "health present is not a heartbeat",
			msg: &protobufs.AgentToServer{ //nolint:exhaustruct
				Health: &protobufs.ComponentHealth{}, //nolint:exhaustruct
			},
			want: false,
		},
		{
			name: "effective config present is not a heartbeat",
			msg: &protobufs.AgentToServer{ //nolint:exhaustruct
				EffectiveConfig: &protobufs.EffectiveConfig{}, //nolint:exhaustruct
			},
			want: false,
		},
		{
			name: "remote config status present is not a heartbeat",
			msg: &protobufs.AgentToServer{ //nolint:exhaustruct
				RemoteConfigStatus: &protobufs.RemoteConfigStatus{}, //nolint:exhaustruct
			},
			want: false,
		},
		{
			name: "package statuses present is not a heartbeat",
			msg: &protobufs.AgentToServer{ //nolint:exhaustruct
				PackageStatuses: &protobufs.PackageStatuses{}, //nolint:exhaustruct
			},
			want: false,
		},
		{
			name: "agent disconnect present is not a heartbeat",
			msg: &protobufs.AgentToServer{ //nolint:exhaustruct
				AgentDisconnect: &protobufs.AgentDisconnect{}, //nolint:exhaustruct
			},
			want: false,
		},
		{
			name: "non-zero flags is not a heartbeat",
			msg: &protobufs.AgentToServer{ //nolint:exhaustruct
				Flags: uint64(protobufs.AgentToServerFlags_AgentToServerFlags_RequestInstanceUid),
			},
			want: false,
		},
		{
			name: "connection settings request present is not a heartbeat",
			msg: &protobufs.AgentToServer{ //nolint:exhaustruct
				ConnectionSettingsRequest: &protobufs.ConnectionSettingsRequest{}, //nolint:exhaustruct
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

// shouldPersistAgentFixture wires up a Service with a stoppable clock and the
// fields shouldPersistAgent reads. The real constructor pulls in too many
// dependencies for a unit test focused on this one decision.
func shouldPersistAgentFixture(now time.Time, throttle time.Duration) *Service {
	return &Service{ //nolint:exhaustruct // only the fields under test are needed
		clock:                 newPersistTestClock(now),
		logger:                slog.Default(),
		heartbeatSaveThrottle: throttle,
	}
}

func TestShouldPersistAgent(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.May, 26, 12, 0, 0, 0, time.UTC)
	throttle := 60 * time.Second

	heartbeat := &protobufs.AgentToServer{}   //nolint:exhaustruct
	nonHeartbeat := &protobufs.AgentToServer{ //nolint:exhaustruct
		Health: &protobufs.ComponentHealth{Healthy: true}, //nolint:exhaustruct
	}

	t.Run("non-heartbeat always persists regardless of lastSaveAt", func(t *testing.T) {
		t.Parallel()

		svc := shouldPersistAgentFixture(now, throttle)
		uid := uuid.New()
		svc.lastSaveAt.Store(uid.String(), now) // freshly saved

		assert.True(t, svc.shouldPersistAgent(uid, nonHeartbeat))
	})

	t.Run("heartbeat with no prior save persists", func(t *testing.T) {
		t.Parallel()

		svc := shouldPersistAgentFixture(now, throttle)

		assert.True(t, svc.shouldPersistAgent(uuid.New(), heartbeat))
	})

	t.Run("heartbeat throttled within the window", func(t *testing.T) {
		t.Parallel()

		svc := shouldPersistAgentFixture(now, throttle)
		uid := uuid.New()
		svc.lastSaveAt.Store(uid.String(), now.Add(-30*time.Second))

		assert.False(t, svc.shouldPersistAgent(uid, heartbeat))
	})

	t.Run("heartbeat persists once the throttle window has elapsed", func(t *testing.T) {
		t.Parallel()

		svc := shouldPersistAgentFixture(now, throttle)
		uid := uuid.New()
		svc.lastSaveAt.Store(uid.String(), now.Add(-throttle))

		assert.True(t, svc.shouldPersistAgent(uid, heartbeat))
	})

	t.Run("corrupt cache entry forces a save", func(t *testing.T) {
		t.Parallel()

		svc := shouldPersistAgentFixture(now, throttle)
		uid := uuid.New()
		svc.lastSaveAt.Store(uid.String(), "not-a-time") // wrong type

		assert.True(t, svc.shouldPersistAgent(uid, heartbeat))
	})
}

// newPersistTestClock returns a fixed clock for the persistence-throttle tests.
// We reuse the existing test clock pattern from server_test.go but keep this
// file self-contained.
func newPersistTestClock(t time.Time) clock.Clock {
	return &persistTestClock{now: t}
}

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
