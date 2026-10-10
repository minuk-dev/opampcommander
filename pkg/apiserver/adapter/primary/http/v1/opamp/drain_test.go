package opamp_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/opamp"
)

func TestController_Drain(t *testing.T) {
	t.Parallel()

	for _, secure := range []bool{false, true} {
		name := "ws"
		if secure {
			name = "wss"
		}

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			controller := opamp.NewController(&spyUsecase{}, slog.Default())

			engine := gin.New()
			for _, route := range controller.RoutesInfo() {
				engine.Handle(route.Method, route.Path, route.HandlerFunc)
			}

			server := httptest.NewUnstartedServer(engine)

			server.Config.ConnContext = controller.ConnContext
			if secure {
				server.StartTLS()
			} else {
				server.Start()
			}
			defer server.Close()

			dialer := websocket.Dialer{}

			if secure {
				transport, ok := server.Client().Transport.(*http.Transport)
				require.True(t, ok)

				dialer.TLSClientConfig = transport.TLSClientConfig
			}

			url := strings.Replace(server.URL, "http", "ws", 1) + "/api/v1/opamp"
			conn, response, err := dialer.DialContext(t.Context(), url, nil)
			require.NoError(t, err)

			require.NoError(t, response.Body.Close())
			defer func() { _ = conn.Close() }()

			require.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))

			drained := make(chan error, 1)

			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()

			go func() { drained <- controller.Drain(ctx, 10*time.Millisecond) }()

			_, _, err = conn.ReadMessage()
			require.True(t, websocket.IsCloseError(err, websocket.CloseGoingAway), "got %v", err)
			require.NoError(t, <-drained)
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+"/api/v1/opamp", nil)
			require.NoError(t, err)
			rejected, err := server.Client().Do(req)
			require.NoError(t, err)

			defer func() { _ = rejected.Body.Close() }()

			assert.Equal(t, http.StatusServiceUnavailable, rejected.StatusCode)
		})
	}
}

func TestController_DrainForceClosesUnresponsivePeer(t *testing.T) {
	t.Parallel()

	controller := opamp.NewController(&spyUsecase{}, slog.Default())
	engine := gin.New()
	engine.GET("/api/v1/opamp", controller.Handle)
	server := httptest.NewUnstartedServer(engine)
	server.Config.ConnContext = controller.ConnContext

	server.Start()
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/opamp"
	conn, response, err := websocket.DefaultDialer.DialContext(t.Context(), url, nil)
	require.NoError(t, err)

	require.NoError(t, response.Body.Close())
	defer func() { _ = conn.Close() }()
	// Do not read or acknowledge the server close until the deadline expires.
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()

	require.ErrorIs(t, controller.Drain(ctx, 0), context.DeadlineExceeded)
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(time.Second)))
	_, _, err = conn.ReadMessage()
	require.True(t, websocket.IsCloseError(err, websocket.CloseGoingAway), "got %v", err)
	_, err = conn.UnderlyingConn().Read(make([]byte, 1))
	require.Error(t, err)

	var netErr interface{ Timeout() bool }
	if errors.As(err, &netErr) {
		assert.False(t, netErr.Timeout(), "socket must be force-closed")
	}
}
