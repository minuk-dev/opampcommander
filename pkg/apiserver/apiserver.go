// Package apiserver provides app logic for the opampcommander apiserver.
package apiserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	applicationmodule "github.com/minuk-dev/opampcommander/pkg/apiserver/module/application"
	domainmodule "github.com/minuk-dev/opampcommander/pkg/apiserver/module/domain"
	eventmodule "github.com/minuk-dev/opampcommander/pkg/apiserver/module/event"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/module/helper"
	inmodule "github.com/minuk-dev/opampcommander/pkg/apiserver/module/in"
	outmodule "github.com/minuk-dev/opampcommander/pkg/apiserver/module/out"
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
	app := fx.New(
		// hexagonal architecture
		inmodule.New(),
		outmodule.New(),
		applicationmodule.New(),
		domainmodule.New(),
		eventmodule.New(),
		NewConfigModule(&settings),

		// base
		helper.NewModule(),
		// init
		fx.Invoke(func(*http.Server) {}),
	)

	server := &Server{
		App:      app,
		settings: settings,
	}

	return server
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

// NewConfigModule creates a new module for configuration.
func NewConfigModule(settings *config.ServerSettings) fx.Option {
	return fx.Module(
		"config",
		// config
		fx.Provide(helper.ValueFunc(settings)),
		fx.Provide(helper.PointerFunc(settings.DatabaseSettings)),
		fx.Provide(helper.PointerFunc(settings.AuthSettings)),
		fx.Provide(helper.PointerFunc(settings.ManagementSettings)),
		fx.Provide(helper.PointerFunc(settings.ManagementSettings.ObservabilitySettings)),
		fx.Provide(helper.PointerFunc(settings.EventSettings)),
		// serverID provider with explicit type
		fx.Provide(func() config.ServerID { return settings.ServerID }),
	)
}
