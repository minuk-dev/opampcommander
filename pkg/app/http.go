package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

const (
	// DefaultHTTPReadTimeout is the default timeout for reading HTTP requests.
	// It should be set to a reasonable value to avoid security issues.
	DefaultHTTPReadTimeout = 30 * time.Second
)

// NewHTTPServer creates a new HTTP server instance.
func NewHTTPServer(
	lifecycle fx.Lifecycle,
	engine *gin.Engine,
	settings *ServerSettings,
	logger *slog.Logger,
	connContext func(context.Context, net.Conn) context.Context,
) *http.Server {
	//exhaustruct:ignore
	srv := &http.Server{
		ReadTimeout: DefaultHTTPReadTimeout,
		Addr:        settings.Addr,
		Handler:     engine,
		ConnContext: connContext,
	}

	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			listener, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				return fmt.Errorf("failed to listen: %w", err)
			}
			logger.Info("HTTP server listening",
				slog.String("addr", settings.Addr),
			)
			go func() {
				err := srv.Serve(listener)
				if err != nil && !errors.Is(err, http.ErrServerClosed) {
					logger.Error("HTTP server error",
						slog.String("error", err.Error()),
					)
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})

	return srv
}
