package apiserver

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	"github.com/minuk-dev/opampcommander/internal/observability"
	"github.com/minuk-dev/opampcommander/internal/security"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
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
		NewInPortModule(),
		NewApplicationServiceModule(),
		NewDomainServiceModule(),
		NewOutPortModule(),
		NewConfigModule(&settings),

		// base
		fx.Provide(
			// executor for runners
			fx.Annotate(NewExecutor, fx.ParamTags("", `group:"runners"`)),
			// logger
			NewLogger,
			// security,
			security.New,
			observability.New,
		),
		// init
		fx.Invoke(func(*http.Server) {}),
		fx.Invoke(func(*Executor) {}),
		fx.WithLogger(func(logger *slog.Logger) fxevent.Logger {
			return &fxevent.SlogLogger{
				Logger: logger,
			}
		}),
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
	stopCtx, stopCancel := context.WithTimeout(NoInheritContext(ctx), DefaultServerStopTimeout)
	defer stopCancel()

	err = s.Stop(stopCtx)
	if err != nil {
		return fmt.Errorf("failed to stop the server: %w", err)
	}

	return nil
}
