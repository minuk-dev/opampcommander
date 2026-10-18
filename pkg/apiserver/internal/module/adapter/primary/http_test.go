package primary_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx/fxtest"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/internal/module/adapter/primary"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/management/observability"
)

func TestNewHTTPServer(t *testing.T) {
	t.Parallel()

	const fixtures = "../../../servertls/testdata/"

	for _, secure := range []bool{false, true} {
		name := "plaintext"
		if secure {
			name = "TLS"
		}

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var listenConfig net.ListenConfig

			listener, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
			require.NoError(t, err)

			settings := &config.ServerSettings{Address: listener.Addr().String()}
			require.NoError(t, listener.Close())

			if secure {
				settings.TLS = config.TLSSettings{CertFile: fixtures + "server.crt", KeyFile: fixtures + "server.key"}
			}

			engine := gin.New()
			engine.GET("/api/v1/ping", func(ctx *gin.Context) { ctx.Status(http.StatusNoContent) })
			engine.GET("/api/v1/opamp", func(ctx *gin.Context) {
				upgrader := websocket.Upgrader{}

				connection, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
				if err == nil {
					_ = connection.Close()
				}
			})

			lifecycle := fxtest.NewLifecycle(t)

			var logs bytes.Buffer

			logger := slog.New(slog.NewTextHandler(&logs, nil))
			_, err = primary.NewHTTPServer(lifecycle, engine, settings, logger,
				func(ctx context.Context, _ net.Conn) context.Context { return ctx })
			require.NoError(t, err)

			lifecycle.RequireStart()
			defer lifecycle.RequireStop()

			roots := x509.NewCertPool()
			ca, err := os.ReadFile(fixtures + "server.crt")
			require.NoError(t, err)
			require.True(t, roots.AppendCertsFromPEM(ca))
			clientTLS := &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}
			client := &http.Client{Transport: &http.Transport{TLSClientConfig: clientTLS}, Timeout: time.Second}
			t.Cleanup(client.CloseIdleConnections)

			scheme, wsScheme := "http://", "ws://"
			if secure {
				scheme, wsScheme = "https://", "wss://"
			}

			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, scheme+settings.Address+"/api/v1/ping", nil)
			require.NoError(t, err)
			response, err := client.Do(req)
			require.NoError(t, err)
			require.NoError(t, response.Body.Close())
			require.Equal(t, http.StatusNoContent, response.StatusCode)

			dialer := websocket.Dialer{TLSClientConfig: clientTLS}
			connection, response, err := dialer.DialContext(t.Context(), wsScheme+settings.Address+"/api/v1/opamp", nil)
			require.NoError(t, err)
			require.NoError(t, response.Body.Close())
			require.NoError(t, connection.Close())

			if secure {
				require.NotContains(t, logs.String(), "TLS disabled")
			} else {
				require.Contains(t, logs.String(), "ws://"+settings.Address+"/api/v1/opamp")
				require.Contains(t, logs.String(), "WARN")
			}
		})
	}
}

func TestNewEngine_Swagger(t *testing.T) {
	t.Parallel()

	engine := primary.NewEngine(nil, nil, nil, nil, &config.ServerSettings{}, &observability.Service{}, slog.Default())

	for _, testCase := range []struct {
		name        string
		path        string
		status      int
		contentType string
		body        string
	}{
		{"redirect", "/docs", http.StatusMovedPermanently, "text/html", "/swagger/index.html"},
		{"index", "/swagger/index.html", http.StatusOK, "text/html", `url: "doc.json"`},
		{"spec", "/swagger/doc.json", http.StatusOK, "application/json", `"swagger": "2.0"`},
		{"css", "/swagger/swagger-ui.css", http.StatusOK, "text/css", ".swagger-ui"},
		{"javascript", "/swagger/swagger-ui-bundle.js", http.StatusOK, "javascript", "SwaggerUIBundle"},
		{"missing asset", "/swagger/missing.js", http.StatusNotFound, "text/plain", "404 page not found"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			response := httptest.NewRecorder()
			engine.ServeHTTP(response, httptest.NewRequestWithContext(t.Context(), http.MethodGet, testCase.path, nil))

			require.Equal(t, testCase.status, response.Code)
			require.Contains(t, response.Header().Get("Content-Type"), testCase.contentType)
			require.Contains(t, response.Body.String(), testCase.body)
		})
	}
}
