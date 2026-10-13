package servertls

import (
	"crypto/tls"
	"crypto/x509"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx/fxtest"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
)

func TestNew(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		settings config.TLSSettings
		valid    bool
	}{
		{"plaintext", config.TLSSettings{}, true},
		{"HTTPS", config.TLSSettings{CertFile: "testdata/server.crt", KeyFile: "testdata/server.key"}, true},
		{"client CA", config.TLSSettings{
			CertFile: "testdata/server.crt", KeyFile: "testdata/server.key", CAFile: "testdata/server.crt",
		}, true},
		{"missing file", config.TLSSettings{CertFile: "missing", KeyFile: "missing"}, false},
		{"mismatched key", config.TLSSettings{CertFile: "testdata/server.crt", KeyFile: "testdata/rotated.key"}, false},
		{"missing CA", config.TLSSettings{
			CertFile: "testdata/server.crt", KeyFile: "testdata/server.key", CAFile: "missing",
		}, false},
		{"invalid CA", config.TLSSettings{
			CertFile: "testdata/server.crt", KeyFile: "testdata/server.key", CAFile: "testdata/server.key",
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := New(tc.settings, fxtest.NewLifecycle(t), slog.Default())
			if !tc.valid {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)

			if tc.settings.CertFile == "" {
				require.Nil(t, got)
			} else if tc.settings.CAFile != "" {
				require.Equal(t, tls.VerifyClientCertIfGiven, got.ClientAuth)
				require.NotNil(t, got.ClientCAs)
			}
		})
	}
}

//nolint:paralleltest,gosec // Process-wide SIGHUP; file paths are test fixtures and t.TempDir only.
func TestNew_Reload(t *testing.T) {
	dir := t.TempDir()
	settings := config.TLSSettings{CertFile: filepath.Join(dir, "server.crt"), KeyFile: filepath.Join(dir, "server.key")}
	copyFile := func(destination, source string) {
		t.Helper()

		data, err := os.ReadFile(source)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(destination, data, 0o600))
	}
	copyFile(settings.CertFile, "testdata/server.crt")
	copyFile(settings.KeyFile, "testdata/server.key")

	lifecycle := fxtest.NewLifecycle(t)
	logger := slog.New(slog.DiscardHandler)
	tlsConfig, err := New(settings, lifecycle, logger)
	require.NoError(t, err)

	lifecycle.RequireStart()
	defer lifecycle.RequireStop()

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	server.Listener = tls.NewListener(server.Listener, tlsConfig)
	server.Start()

	server.URL = strings.Replace(server.URL, "http://", "https://", 1)
	defer server.Close()

	roots := x509.NewCertPool()

	for _, path := range []string{"testdata/server.crt", "testdata/rotated.crt"} {
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		require.True(t, roots.AppendCertsFromPEM(data))
	}

	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}, DisableKeepAlives: true,
	}, Timeout: time.Second}
	t.Cleanup(client.CloseIdleConnections)

	serial := func() int64 {
		t.Helper()

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
		require.NoError(t, err)
		response, err := client.Do(req)
		require.NoError(t, err)
		require.NoError(t, response.Body.Close())

		return response.TLS.PeerCertificates[0].SerialNumber.Int64()
	}
	require.EqualValues(t, 1, serial())

	copyFile(settings.CertFile, "testdata/rotated.crt")
	copyFile(settings.KeyFile, "testdata/rotated.key")
	require.NoError(t, syscall.Kill(os.Getpid(), syscall.SIGHUP))
	require.Eventually(t, func() bool {
		certificate, err := tlsConfig.GetCertificate(nil)

		return err == nil && certificate.Leaf.SerialNumber.Int64() == 2
	}, time.Second, time.Millisecond)
	require.EqualValues(t, 2, serial())

	// An incomplete or mismatched rotation cannot replace the last valid pair.
	copyFile(settings.KeyFile, "testdata/server.key")
	store := &certificateStore{settings: settings}
	certificate, err := tlsConfig.GetCertificate(nil)
	require.NoError(t, err)
	store.current.Store(certificate)
	require.Error(t, store.reload())
	require.Same(t, certificate, store.current.Load())
	require.EqualValues(t, 2, serial())
}
