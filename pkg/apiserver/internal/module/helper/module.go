package helper

import (
	"go.uber.org/fx"

	internalhelper "github.com/minuk-dev/opampcommander/pkg/apiserver/helper"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/internal/module/helper/lifecycle"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/security"
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
