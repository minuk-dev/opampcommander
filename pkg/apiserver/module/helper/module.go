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
			// Lifecycle management
			fx.Annotate(lifecycle.NewExecutor, fx.ParamTags(``, `group:"runners"`)),
			internalhelper.NewShutdownListener,
			// Security
			security.New,
			// Management
		),
		// Lifecycle hooks
		fx.Invoke(func(*lifecycle.Executor) {}),
		fx.Invoke(lifecycle.RegisterShutdownListener),
	)
}
