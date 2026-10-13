package management_test

import (
	"crypto/tls"
	"crypto/x509"
	"log/slog"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx/fxtest"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/internal/module/helper/management"
)

func TestNewHTTPServer(t *testing.T) {
	t.Parallel()

	const fixtures = "../../../servertls/testdata/"

	for _, mode := range []string{"plaintext", "TLS", "mTLS"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()

			var listenConfig net.ListenConfig

			listener, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
			require.NoError(t, err)

			settings := &config.ManagementSettings{Address: listener.Addr().String()}
			require.NoError(t, listener.Close())

			if mode != "plaintext" {
				settings.TLS = config.TLSSettings{CertFile: fixtures + "server.crt", KeyFile: fixtures + "server.key"}
			}

			if mode == "mTLS" {
				settings.TLS.CAFile = fixtures + "server.crt"
			}

			lifecycle := fxtest.NewLifecycle(t)
			_, err = management.NewHTTPServer(settings, nil, lifecycle, slog.Default())
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

			scheme := "https://"
			if mode == "plaintext" {
				scheme = "http://"
			}

			request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, scheme+settings.Address+"/healthz", nil)
			require.NoError(t, err)

			if mode == "mTLS" {
				response, err := client.Do(request)
				if response != nil {
					require.NoError(t, response.Body.Close())
				}

				require.Error(t, err, "management CA must require client authentication")
				certificate, err := tls.LoadX509KeyPair(fixtures+"server.crt", fixtures+"server.key")
				require.NoError(t, err)

				clientTLS.Certificates = []tls.Certificate{certificate}
			}

			response, err := client.Do(request)
			require.NoError(t, err)
			require.NoError(t, response.Body.Close())
			require.Equal(t, http.StatusNotFound, response.StatusCode, "TLS request reached the empty management mux")
		})
	}
}

func TestNewHTTPServer_InvalidTLS(t *testing.T) {
	t.Parallel()

	_, err := management.NewHTTPServer(&config.ManagementSettings{TLS: config.TLSSettings{CertFile: "missing"}},
		nil, fxtest.NewLifecycle(t), slog.Default())
	require.ErrorIs(t, err, config.ErrInvalidTLS)
}

func TestNewHTTPServer_AddressInUse(t *testing.T) {
	t.Parallel()

	var listenConfig net.ListenConfig

	listener, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")

	require.NoError(t, err)
	defer func() { require.NoError(t, listener.Close()) }()

	lifecycle := fxtest.NewLifecycle(t)
	_, err = management.NewHTTPServer(&config.ManagementSettings{Address: listener.Addr().String()},
		nil, lifecycle, slog.Default())
	require.NoError(t, err)
	require.ErrorContains(t, lifecycle.Start(t.Context()), "listen on management address")
}
