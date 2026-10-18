package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
)

func TestShutdownSettings(t *testing.T) {
	t.Parallel()

	defaults := config.ShutdownSettings{}.WithDefaults()
	assert.Equal(t, 10*time.Second, defaults.DrainWindow)
	assert.Equal(t, 30*time.Second, defaults.Timeout)

	for _, tc := range []struct {
		name              string
		window, timeLimit time.Duration
		valid             bool
	}{
		{"defaults", 0, 0, true},
		{"custom", time.Second, 2 * time.Second, true},
		{"negative window", -time.Second, time.Second, false},
		{"negative timeout", time.Second, -time.Second, false},
		{"no time for handshakes", time.Second, time.Second, false},
		{"window exceeds timeout", 2 * time.Second, time.Second, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			settings := config.ShutdownSettings{DrainWindow: tc.window, Timeout: tc.timeLimit}
			if tc.valid {
				require.NoError(t, settings.Validate())
			} else {
				require.ErrorIs(t, settings.Validate(), config.ErrInvalidShutdown)
			}
		})
	}
}
