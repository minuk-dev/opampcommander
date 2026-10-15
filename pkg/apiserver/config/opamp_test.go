package config_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
)

func TestOpAMPSettings(t *testing.T) {
	t.Parallel()
	require.Equal(t, config.OpAMPSettings{}, config.DefaultOpAMPSettings())

	for _, tt := range []struct {
		name  string
		limit int64
	}{
		{"default", 0}, {"positive", 1024}, {"unlimited", -1},
		{"minimum", math.MinInt64}, {"maximum", math.MaxInt64},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			settings := config.OpAMPSettings{MaxMessageSize: tt.limit}
			require.NoError(t, settings.Validate())
		})
	}
}
