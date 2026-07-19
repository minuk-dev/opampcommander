// Package app is the apiserver composition root: it wires the FX modules and
// owns the application lifecycle. It is internal because it is the only place
// allowed to depend on Uber FX; the public github.com/.../pkg/apiserver package
// is a thin, FX-free wrapper over it.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	adaptermodule "github.com/minuk-dev/opampcommander/pkg/apiserver/internal/module/adapter"
	applicationmodule "github.com/minuk-dev/opampcommander/pkg/apiserver/internal/module/application"
	domainmodule "github.com/minuk-dev/opampcommander/pkg/apiserver/internal/module/domain"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/internal/module/helper"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/internal/module/helper/management"
	infrastructuremodule "github.com/minuk-dev/opampcommander/pkg/apiserver/internal/module/infrastructure"
)

const (
	// DefaultServerStartTimeout = 30 * time.Second.
	DefaultServerStartTimeout = 30 * time.Second

	// DefaultServerStopTimeout is the default timeout for stopping the server.
	DefaultServerStopTimeout = 30 * time.Second
)

// Server is a struct that represents the server application.
// It embeds the fx.App struct from the Uber Fx framework.
type Server struct {
	*fx.App

	settings config.ServerSettings
}

// New creates a new instance of the Server struct.
func New(settings config.ServerSettings) *Server {
	app := fx.New(appOptions(&settings)...)

	server := &Server{
		App:      app,
		settings: settings,
	}

	return server
}

// appOptions returns the full FX option set for the apiserver. It is the single source of
// truth for the composition root so New and ValidateWiring stay in sync.
func appOptions(settings *config.ServerSettings) []fx.Option {
	return []fx.Option{
		// Hexagonal architecture layers
		adaptermodule.NewAdapterModules(settings.DatabaseSettings.Type), // Adapters: HTTP, DB, messaging, scheduler
		infrastructuremodule.New(settings.DatabaseSettings.Type),        // Bootstrap: Casbin RBAC + default seed hooks
		applicationmodule.New(),   // Application services
		domainmodule.New(),        // Domain services
		NewConfigModule(settings), // Configuration

		// Base utilities
		helper.NewModule(),
		management.NewModule(),

		// Route FX's own lifecycle events (provided/invoked/OnStart/OnStop) through the
		// app's slog logger instead of FX's default console logger. This makes those logs
		// structured and, crucially, level-controlled — raising the configured log level
		// (e.g. to warn in e2e tests) silences the otherwise very noisy "[Fx] HOOK ..."
		// startup stream.
		fx.WithLogger(func(logger *slog.Logger) fxevent.Logger {
			return &fxevent.SlogLogger{Logger: logger}
		}),

		// Initialize HTTP server
		fx.Invoke(func(*http.Server) {}),
	}
}

// ValidateWiring checks that the apiserver's FX dependency graph resolves (no missing
// dependencies or cycles) without constructing the application or running any lifecycle
// hooks. It is exposed so a fast wiring test can guard against regressions without
// importing FX directly.
func ValidateWiring(settings config.ServerSettings) error {
	//nolint:wrapcheck // thin pass-through to fx.ValidateApp for tests
	return fx.ValidateApp(appOptions(&settings)...)
}

// Run starts the server and blocks until the context is done.
func (s *Server) Run(ctx context.Context) error {
	startCtx, startCancel := context.WithTimeout(ctx, DefaultServerStartTimeout)
	defer startCancel()

	err := s.Start(startCtx)
	if err != nil {
		return fmt.Errorf("failed to start the server: %w", err)
	}

	<-ctx.Done()

	// To gracefully shutdown, it needs stopCtx.
	stopCtx, stopCancel := context.WithTimeout(context.Background(), DefaultServerStopTimeout)
	defer stopCancel()

	err = s.Stop(stopCtx) //nolint:contextcheck
	if err != nil {
		return fmt.Errorf("failed to stop the server: %w", err)
	}

	return nil
}

// VisualizeError renders an FX dependency-graph error into a human-readable form.
// It is exposed so callers outside the composition root can pretty-print startup
// failures without importing FX directly.
func VisualizeError(err error) (string, error) {
	//nolint:wrapcheck // thin pass-through to fx.VisualizeError
	return fx.VisualizeError(err)
}

// NewConfigModule creates a new module for configuration.
func NewConfigModule(settings *config.ServerSettings) fx.Option {
	return fx.Module(
		"config",
		// config
		fx.Provide(helper.ValueFunc(settings)),
		fx.Provide(helper.PointerFunc(settings.DatabaseSettings)),
		// security owns its config; the aggregate composes it and we inject it back.
		fx.Provide(helper.PointerFunc(settings.Security)),
		fx.Provide(helper.PointerFunc(settings.ManagementSettings)),
		// observability owns its config; aggregated under ManagementSettings.
		fx.Provide(helper.PointerFunc(settings.ManagementSettings.Observability)),
		fx.Provide(helper.PointerFunc(settings.EventSettings)),
		// serverID provider with explicit type (owned by the domain)
		fx.Provide(func() agentmodel.ServerID { return settings.ServerID }),
	)
}
