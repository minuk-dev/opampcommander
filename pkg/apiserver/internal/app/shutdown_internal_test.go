package app

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/open-telemetry/opamp-go/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx" //nolint:depguard // Composition-root tests exercise FX lifecycle hooks.

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/opamp"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/usecase"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/management/healthcheck"
)

type noopOpAMP struct {
	usecase.OpAMPUsecase
}

func (*noopOpAMP) OnConnectedWithType(context.Context, types.Connection, bool) {}
func (*noopOpAMP) OnConnectionClose(types.Connection)                          {}
func (*noopOpAMP) OnReadMessageError(types.Connection, int, []byte, error)     {}

func TestOpAMPShutdownLifecycle(t *testing.T) {
	t.Parallel()

	for name, acknowledge := range map[string]bool{"close handshake": true, "unresponsive peer": false} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			controller := opamp.NewController(&noopOpAMP{}, slog.Default())
			health := healthcheck.NewHealthHelper(nil)
			engine := gin.New()
			engine.GET("/api/v1/opamp", controller.Handle)

			settings := &config.ServerSettings{
				Shutdown: config.ShutdownSettings{DrainWindow: time.Millisecond, Timeout: time.Second},
			}
			server := httptest.NewUnstartedServer(engine)

			server.Config.ConnContext = controller.ConnContext
			defer server.Close()

			application := fx.New(
				fx.NopLogger,
				fx.Supply(controller, health, settings),
				fx.Provide(func(lifecycle fx.Lifecycle) *http.Server {
					lifecycle.Append(fx.Hook{
						OnStart: func(context.Context) error {
							server.Start()

							return nil
						},
						OnStop: server.Config.Shutdown,
					})

					return server.Config
				}),
				fx.Invoke(registerOpAMPShutdown),
			)
			require.NoError(t, application.Start(t.Context()))

			url := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/opamp"
			conn, response, err := websocket.DefaultDialer.DialContext(t.Context(), url, nil)
			require.NoError(t, err)

			require.NoError(t, response.Body.Close())
			defer func() { _ = conn.Close() }()

			require.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))

			stopped := make(chan error, 1)
			go func() { stopped <- application.Stop(t.Context()) }()

			if acknowledge {
				conn.SetCloseHandler(func(code int, text string) error {
					ready, reasons := health.Readiness(t.Context())
					assert.False(t, ready, "readiness must be false before the close frame")
					assert.Contains(t, reasons, "shutdown")
					assert.Equal(t, websocket.CloseGoingAway, code)
					// HTTP must still be serving while the close handshake is in progress.
					req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+"/api/v1/opamp", nil)
					require.NoError(t, err)
					response, err := server.Client().Do(req)
					require.NoError(t, err)
					require.NoError(t, response.Body.Close())
					assert.Equal(t, http.StatusServiceUnavailable, response.StatusCode)

					err = conn.WriteControl(
						websocket.CloseMessage, websocket.FormatCloseMessage(code, text), time.Now().Add(time.Second),
					)
					require.NoError(t, err)

					return nil
				})
			} else {
				require.ErrorIs(t, <-stopped, context.DeadlineExceeded)
			}

			_, _, err = conn.ReadMessage()
			require.True(t, websocket.IsCloseError(err, websocket.CloseGoingAway), "got %v", err)

			if acknowledge {
				require.NoError(t, <-stopped)
			}
			// HTTP shuts down after the handshake, or is force-closed at the deadline.
			require.ErrorIs(t, server.Config.ListenAndServe(), http.ErrServerClosed)
		})
	}
}
