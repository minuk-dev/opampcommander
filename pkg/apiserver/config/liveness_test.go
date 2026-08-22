package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
)

func TestDefaultLivenessSettingsValidate(t *testing.T) {
	t.Parallel()

	// The shipped defaults must pass their own validation, or every server that
	// configures nothing would refuse to start.
	require.NoError(t, config.DefaultLivenessSettings().Validate())
}

func TestMaxLivenessStalenessBudgetLeavesHeadroom(t *testing.T) {
	t.Parallel()

	assert.Less(t, config.MaxLivenessStalenessBudget(), agentmodel.DefaultConnectionStaleness,
		"a cadence at the staleness window would let live agents read as disconnected")

	settings := config.DefaultLivenessSettings()
	assert.LessOrEqual(t,
		settings.EffectiveFlushInterval()+settings.EffectiveFlushStaleAfter(),
		config.MaxLivenessStalenessBudget(),
		"the shipped defaults must fit their own budget")
	assert.LessOrEqual(t, settings.EffectivePersistThrottle(), config.MaxLivenessStalenessBudget())
}

func TestLivenessSettingsValidate_RejectsAnOversizedCadence(t *testing.T) {
	t.Parallel()

	settings := config.DefaultLivenessSettings()
	settings.FlushInterval = agentmodel.DefaultConnectionStaleness

	require.ErrorIs(t, settings.Validate(), config.ErrLivenessCadenceTooSlow)
}

func TestLivenessSettingsValidate_JudgesTheCadenceAsAPair(t *testing.T) {
	t.Parallel()

	// Each half fits the budget on its own; together they do not. Bounding only the
	// interval — as an earlier version of this did — accepts exactly this, and lets a
	// stored document age past the staleness window while the config looks correct.
	settings := config.DefaultLivenessSettings()
	settings.FlushInterval = config.MaxLivenessStalenessBudget()
	settings.FlushStaleAfter = config.MaxLivenessStalenessBudget()

	require.ErrorIs(t, settings.Validate(), config.ErrLivenessCadenceTooSlow)
}

func TestLivenessSettingsValidate_RedisRequiresEndpoints(t *testing.T) {
	t.Parallel()

	settings := config.DefaultLivenessSettings()
	settings.Redis.Enabled = true

	err := settings.Validate()
	require.ErrorIs(t, err, config.ErrLivenessRedisNoEndpoints)
}

func TestLivenessSettingsValidate_RedisTTLMustOutliveTheStalenessWindow(t *testing.T) {
	t.Parallel()

	settings := config.DefaultLivenessSettings()
	settings.Redis.Enabled = true
	settings.Redis.Endpoints = []string{"localhost:6379"}
	settings.Redis.TTL = agentmodel.DefaultConnectionStaleness

	err := settings.Validate()
	require.ErrorIs(t, err, config.ErrLivenessRedisTTLTooShort)
}

func TestLivenessSettingsValidate_RejectsNegativeTimeouts(t *testing.T) {
	t.Parallel()

	settings := config.DefaultLivenessSettings()
	settings.Redis.Enabled = true
	settings.Redis.Endpoints = []string{"localhost:6379"}
	settings.Redis.CommandTimeout = -time.Second

	err := settings.Validate()
	require.ErrorIs(t, err, config.ErrLivenessRedisTimeoutInvalid)
}

func TestLivenessSettingsValidate_IgnoresRedisWhenDisabled(t *testing.T) {
	t.Parallel()

	// A leftover, nonsensical Redis block must not block startup while the tier is
	// switched off — that would make disabling it useless as an escape hatch.
	settings := config.DefaultLivenessSettings()
	settings.Redis.Enabled = false
	settings.Redis.Endpoints = nil
	settings.Redis.TTL = time.Millisecond

	require.NoError(t, settings.Validate())
}

func TestEffectivePersistThrottle(t *testing.T) {
	t.Parallel()

	t.Run("no shared tier keeps the throttle short", func(t *testing.T) {
		t.Parallel()

		settings := config.DefaultLivenessSettings()

		assert.Equal(t, 10*time.Second, settings.EffectivePersistThrottle())
	})

	t.Run("a shared tier hands the routine write path to the flush", func(t *testing.T) {
		t.Parallel()

		settings := config.DefaultLivenessSettings()
		settings.Redis.Enabled = true

		assert.Equal(t, config.MaxLivenessStalenessBudget(), settings.EffectivePersistThrottle())
	})

	t.Run("an explicit value overrides both", func(t *testing.T) {
		t.Parallel()

		settings := config.DefaultLivenessSettings()
		settings.Redis.Enabled = true
		settings.PersistThrottle = 3 * time.Second

		assert.Equal(t, 3*time.Second, settings.EffectivePersistThrottle())
	})

	t.Run("an explicit value is still capped at the budget", func(t *testing.T) {
		t.Parallel()

		// The throttle is how long an agent may keep heartbeating without its stored
		// document being refreshed, so an unbounded one has the same effect as an
		// overlong flush cadence.
		settings := config.DefaultLivenessSettings()
		settings.PersistThrottle = time.Hour

		assert.Equal(t, config.MaxLivenessStalenessBudget(), settings.EffectivePersistThrottle())
	})
}

func TestEffectiveRedisDefaults(t *testing.T) {
	t.Parallel()

	//exhaustruct:ignore
	empty := config.RedisLivenessSettings{}

	assert.Equal(t, 2*time.Second, empty.EffectiveDialTimeout())
	assert.Equal(t, 200*time.Millisecond, empty.EffectiveCommandTimeout())
	assert.Greater(t, empty.EffectiveTTL(), agentmodel.DefaultConnectionStaleness)
}
