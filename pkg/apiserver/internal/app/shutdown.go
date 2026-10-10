package app

import (
	"context"
	"fmt"
	"net/http"

	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/management/healthcheck"
)

// Depend on HTTP so its hook is registered first; FX stops hooks in reverse order.
func registerConnectionShutdown(
	lifecycle fx.Lifecycle,
	service *agentservice.Service,
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

			err := service.Shutdown(ctx, settings.Shutdown.DrainWindow)
			if ctx.Err() != nil {
				// FX skips later hooks after its stop deadline expires.
				_ = server.Close()
			}

			if err != nil {
				return fmt.Errorf("shutdown connection service: %w", err)
			}

			return nil
		},
	})
}
