package primary_test

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

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/common/opampconnection"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/opamp"
	connectionstore "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/store/inmemory"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/connectionshutdown"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/usecase"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/internal/module/adapter/primary"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/management/healthcheck"
)

type lifecycle struct{ hooks []fx.Hook }

func (l *lifecycle) Append(hook fx.Hook) { l.hooks = append(l.hooks, hook) }

type noopOpAMP struct {
	usecase.OpAMPUsecase

	store agentport.ConnectionStore
}

func (s *noopOpAMP) OnConnectedWithType(ctx context.Context, conn types.Connection, _ bool) {
	_ = s.store.Put(ctx, agentmodel.NewConnection(conn, agentmodel.ConnectionTypeWebSocket))
}
func (*noopOpAMP) OnConnectionClose(types.Connection)                      {}
func (*noopOpAMP) OnReadMessageError(types.Connection, int, []byte, error) {}

func TestHTTPServer_ShutdownDrainsBeforeHTTPShutdown(t *testing.T) {
	t.Parallel()

	store := connectionstore.NewConnectionStore()
	transport := opampconnection.NewTransport()
	controller := opamp.NewController(&noopOpAMP{store: store}, slog.Default(), transport)
	shutdown := connectionshutdown.New(agentservice.NewConnectionShutdownService(store, transport))
	health := healthcheck.NewHealthHelper(nil)
	engine := gin.New()
	engine.GET("/api/v1/opamp", controller.Handle)

	hooks := &lifecycle{}
	settings := &config.ServerSettings{
		Shutdown: config.ShutdownSettings{DrainWindow: time.Millisecond, Timeout: time.Second},
	}
	srv, err := primary.NewHTTPServer(hooks, engine, settings, slog.Default(), controller.ConnContext, shutdown, health)
	require.NoError(t, err)

	server := httptest.NewUnstartedServer(srv.Handler)
	server.Config.ConnContext = srv.ConnContext

	server.Start()
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/opamp"
	conn, response, err := websocket.DefaultDialer.DialContext(t.Context(), url, nil)
	require.NoError(t, err)

	require.NoError(t, response.Body.Close())
	defer func() { _ = conn.Close() }()

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(time.Second)))

	stopped := make(chan error, 1)
	go func() { stopped <- hooks.hooks[0].OnStop(t.Context()) }()

	conn.SetCloseHandler(func(code int, text string) error {
		ready, reasons := health.Readiness(t.Context())
		assert.False(t, ready, "readiness must be false before the close frame")
		assert.Contains(t, reasons, "shutdown")
		assert.Equal(t, websocket.CloseGoingAway, code)
		err := conn.WriteControl(
			websocket.CloseMessage, websocket.FormatCloseMessage(code, text), time.Now().Add(time.Second),
		)
		require.NoError(t, err)

		return nil
	})
	_, _, err = conn.ReadMessage()
	require.True(t, websocket.IsCloseError(err, websocket.CloseGoingAway), "got %v", err)
	require.NoError(t, <-stopped)
	// Shutdown has run only after the WebSocket close handshake completed.
	require.ErrorIs(t, srv.ListenAndServe(), http.ErrServerClosed)
}
