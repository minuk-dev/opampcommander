// Package lifecycle provides application lifecycle wiring, such as the graceful
// shutdown listener hook.
package lifecycle

import (
	"context"
	"fmt"

	"go.uber.org/fx"

	internalhelper "github.com/minuk-dev/opampcommander/pkg/apiserver/helper"
)

// RegisterShutdownListener registers the shutdown listener in the lifecycle.
func RegisterShutdownListener(
	shutdownlistener *internalhelper.ShutdownListener,
	lifecycle fx.Lifecycle,
) {
	lifecycle.Append(fx.Hook{
		OnStart: nil,
		OnStop: func(ctx context.Context) error {
			err := shutdownlistener.Shutdown(ctx)
			if err != nil {
				return fmt.Errorf("error during shutdown: %w", err)
			}

			return nil
		},
	})
}
