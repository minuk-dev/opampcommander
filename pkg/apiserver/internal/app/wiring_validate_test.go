package app_test

import (
	"testing"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/internal/app"
)

// TestWiringValidates checks the full FX dependency graph resolves without missing
// dependencies or cycles — a fast guard (in-memory DB, no lifecycle/network) against
// wiring regressions such as the generic reconcile registry pulling in a cycle.
func TestWiringValidates(t *testing.T) {
	t.Parallel()

	//exhaustruct:ignore
	settings := config.ServerSettings{
		DatabaseSettings: config.DatabaseSettings{Type: config.DatabaseTypeInMemory},
	}

	err := app.ValidateWiring(settings)
	if err != nil {
		t.Fatalf("fx app graph failed to validate: %v", err)
	}
}
