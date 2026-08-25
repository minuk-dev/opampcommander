package redis // simulating a TTL expiry means deleting the record key behind the store's back

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	redisTestContainer "github.com/testcontainers/testcontainers-go/modules/redis"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
)

func TestStorePendingSelfHealsAnOrphanedIndexEntry(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping redis container test in short mode")
	}

	// The container is started directly rather than through pkg/testutil: that
	// package pulls in the whole apiserver, which imports this one.
	container, err := redisTestContainer.Run(t.Context(), "redis:7.4-alpine")
	require.NoError(t, err)

	endpoint, err := container.Endpoint(t.Context(), "")
	require.NoError(t, err)

	//exhaustruct:ignore
	store, err := New(Config{
		Endpoints:      []string{endpoint},
		DialTimeout:    2 * time.Second,
		CommandTimeout: 2 * time.Second,
		TTL:            2 * time.Minute,
	})
	require.NoError(t, err)

	t.Cleanup(func() { _ = store.Close() })

	instanceUID := uuid.New()
	now := time.Now()

	observation := &agentmodel.AgentLiveness{
		InstanceUID:       instanceUID,
		Connected:         true,
		ConnectionType:    agentmodel.ConnectionTypeWebSocket,
		SequenceNum:       1,
		LastReportedAt:    now,
		LastReportedTo:    "server-a",
		DurableReportedAt: time.Time{},
	}

	_, err = store.Touch(t.Context(), observation)
	require.NoError(t, err)
	require.NoError(t, store.MarkPersisted(t.Context(), instanceUID, now.Add(-10*time.Minute)))

	_, err = store.Touch(t.Context(), observation)
	require.NoError(t, err)

	// Drop the record the way a TTL expiry would, leaving the index entry behind.
	require.NoError(t, store.client.Del(t.Context(), store.recordKey(instanceUID)).Err())

	pending, err := store.ListPendingWriteThrough(t.Context(), now, 0)
	require.NoError(t, err)
	assert.Empty(t, pending)

	// The orphaned entry must be swept, or the index would grow without bound as
	// agents disappear — nothing else removes them.
	size, err := store.client.ZCard(t.Context(), store.pendingKey).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), size)
}

// TestDecodeRecordRejectsHostileFields pins the decoding of a hash the server did
// not write.
//
// A Redis hash is external input — another deployment sharing the instance, a stale
// key, a hand-edited value — so decoding must not narrow a 64-bit value into a
// smaller type, and must not turn an unparseable field into a plausible-looking one.
// Every bad field falls back to its zero value, which every consumer already treats
// as "not known".
func TestDecodeRecordRejectsHostileFields(t *testing.T) {
	t.Parallel()

	instanceUID := uuid.New()

	cases := []struct {
		name   string
		fields map[string]string
		assert func(t *testing.T, record *agentmodel.AgentLiveness)
	}{
		{
			name:   "connection type beyond the enum",
			fields: map[string]string{fieldConnectionType: "999999999999999999999"},
			assert: func(t *testing.T, record *agentmodel.AgentLiveness) {
				t.Helper()
				assert.Equal(t, agentmodel.ConnectionTypeUnknown, record.ConnectionType)
			},
		},
		{
			name:   "connection type that is not a name at all",
			fields: map[string]string{fieldConnectionType: "\x00garbage"},
			assert: func(t *testing.T, record *agentmodel.AgentLiveness) {
				t.Helper()
				assert.Equal(t, agentmodel.ConnectionTypeUnknown, record.ConnectionType)
			},
		},
		{
			name:   "negative sequence number",
			fields: map[string]string{fieldSequenceNum: "-1"},
			assert: func(t *testing.T, record *agentmodel.AgentLiveness) {
				t.Helper()
				assert.Equal(t, uint64(0), record.SequenceNum,
					"a negative value must not wrap into a huge sequence number")
			},
		},
		{
			name:   "sequence number above MaxInt64 still round-trips",
			fields: map[string]string{fieldSequenceNum: "18446744073709551615"},
			assert: func(t *testing.T, record *agentmodel.AgentLiveness) {
				t.Helper()
				assert.Equal(t, uint64(math.MaxUint64), record.SequenceNum,
					"SequenceNum is a uint64; routing it through int64 would lose the top half of the range")
			},
		},
		{
			name:   "unparseable timestamp",
			fields: map[string]string{fieldLastReportedAt: "not-a-number"},
			assert: func(t *testing.T, record *agentmodel.AgentLiveness) {
				t.Helper()
				assert.True(t, record.LastReportedAt.IsZero())
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			record := decodeRecord(instanceUID, testCase.fields)
			require.NotNil(t, record)
			testCase.assert(t, record)
		})
	}
}

// TestConnectionTypeRoundTrip pins the encoding both adapters share.
func TestConnectionTypeRoundTrip(t *testing.T) {
	t.Parallel()

	for _, connectionType := range []agentmodel.ConnectionType{
		agentmodel.ConnectionTypeUnknown,
		agentmodel.ConnectionTypeHTTP,
		agentmodel.ConnectionTypeWebSocket,
	} {
		//exhaustruct:ignore
		encoded := encodeObservation(&agentmodel.AgentLiveness{ConnectionType: connectionType})

		fields := make(map[string]string, len(encoded)/2)

		for i := 0; i < len(encoded); i += 2 {
			name, ok := encoded[i].(string)
			require.True(t, ok, "field names are written as strings")

			fields[name] = fmt.Sprint(encoded[i+1])
		}

		assert.Equal(t, connectionType, decodeRecord(uuid.New(), fields).ConnectionType)
	}
}
