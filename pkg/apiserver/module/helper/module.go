package helper

import (
	"go.uber.org/fx"

	internalhelper "github.com/minuk-dev/opampcommander/internal/helper"
	"github.com/minuk-dev/opampcommander/internal/security"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/module/helper/lifecycle"
)

// NewModule creates a new module for helper services.
func NewModule() fx.Option {
	return fx.Module(
		"helper",
		fx.Provide(
			// Shutdown listener
			internalhelper.NewShutdownListener,
			// Security
			security.New,
		),
		// Lifecycle hooks
		fx.Invoke(lifecycle.RegisterShutdownListener),
	)
}
