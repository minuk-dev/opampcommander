package secondary_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/fx/fxtest"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/internal/module/adapter/secondary"
)

func TestMongoConfigErrorDoesNotExposeURIFragments(t *testing.T) {
	t.Parallel()

	settings := &config.ServerSettings{DatabaseSettings: config.DatabaseSettings{
		Endpoints: []string{"mongodb://user:private-password@localhost/?authSource=%private-secret"},
	}}
	_, err := secondary.NewMongoDBClient(settings, noop.NewTracerProvider(), fxtest.NewLifecycle(t))
	require.ErrorIs(t, err, secondary.ErrInvalidMongoConfig)
	require.NotContains(t, err.Error(), "private-password")
	require.NotContains(t, err.Error(), "private-secret")
}
