// Package management provides management HTTP server functionality.
package management

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/internal/management"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
)

const (
	// DefaultPrometheusReadTimeout is the default timeout for reading Prometheus metrics.
	DefaultPrometheusReadTimeout = 30 * time.Second

	// DefaultPrometheusReadHeaderTimeout is the default timeout for reading Prometheus headers.
	DefaultPrometheusReadHeaderTimeout = 10 * time.Second
)

// HTTPServer provides an HTTP server for management endpoints.
type HTTPServer struct {
	server *http.Server
}

// NewHTTPServer creates a new management HTTP server.
func NewHTTPServer(
	settings *config.ManagementSettings,
	handlers []management.HTTPHandler,
	lifecycle fx.Lifecycle,
	logger *slog.Logger,
) *HTTPServer {
	mux := http.NewServeMux()

	for _, handler := range handlers {
		routes := handler.RoutesInfos()
		for _, route := range routes {
			mux.Handle(route.Path, route.Handler)
			logger.Info("Registered management HTTP route",
				slog.String("path", route.Path),
				slog.String("method", route.Method),
			)
		}
	}

	//exhaustruct:ignore
	server := &http.Server{
		Addr:              settings.Address,
		Handler:           mux,
		ReadHeaderTimeout: DefaultPrometheusReadHeaderTimeout,
	}

	setupMetricsLifecycleHooks(lifecycle, server, logger)

	return &HTTPServer{
		server: server,
	}
}

func setupMetricsLifecycleHooks(
	lifecycle fx.Lifecycle,
	server *http.Server,
	logger *slog.Logger,
) {
	var httpWg sync.WaitGroup

	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			httpWg.Go(func() {
				err := server.ListenAndServe()
				if err != nil && !errors.Is(err, http.ErrServerClosed) {
					logger.Warn("Failed to start Prometheus metrics server", slog.String("error", err.Error()))
				}
			})

			return nil
		},
		OnStop: func(ctx context.Context) error {
			err := server.Shutdown(ctx)
			if err != nil {
				return fmt.Errorf("failed to shutdown Prometheus metrics server: %w", err)
			}

			httpWg.Wait()

			return nil
		},
	})
}
