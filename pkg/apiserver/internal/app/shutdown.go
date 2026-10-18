package app

import (
	"context"
	"fmt"
	"net/http"

	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/opamp"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/management/healthcheck"
)

// Depend on HTTP so its hook is registered first; FX stops hooks in reverse order.
func registerOpAMPShutdown(
	lifecycle fx.Lifecycle,
	controller *opamp.Controller,
	server *http.Server,
	health *healthcheck.HealthHelper,
	settings *config.ServerSettings,
) {
	lifecycle.Append(fx.Hook{
		OnStart: nil,
		OnStop: func(ctx context.Context) error {
			ctx, cancel := context.WithTimeout(ctx, settings.Shutdown.Timeout)
			defer cancel()

			health.BeginShutdown()

			err := controller.Shutdown(ctx, settings.Shutdown.DrainWindow)
			if ctx.Err() != nil {
				// FX skips later hooks after its stop deadline expires.
				_ = server.Close()
			}

			if err != nil {
				return fmt.Errorf("shutdown OpAMP controller: %w", err)
			}

			return nil
		},
	})
}
