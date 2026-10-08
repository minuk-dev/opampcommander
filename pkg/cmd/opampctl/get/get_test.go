package get_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/get"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

func TestGetLookups(t *testing.T) {
	t.Parallel()

	for _, kind := range []string{
		"agent", "agentgroup", "agentpackage", "agentremoteconfig", "application",
		"certificate", "connection", "container", "endpoint", "host", "namespace",
		"remoteconfigschema", "role", "rolebinding", "user",
	} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()

			invalid := ""
			invalidError := "identity must not be empty"

			if kind == "agent" || kind == "connection" {
				invalid = "invalid-uid"
				invalidError = "invalid UID"
			}

			tests := []lookupCase{
				{name: "success", ids: []string{good}, wantItems: 1, wantRequests: []string{good}},
				{name: "invalid identity", ids: []string{invalid}, wantError: invalidError},
				{name: "all not found", ids: []string{missing}, status: http.StatusNotFound,
					wantRequests: []string{missing}, wantError: missing},
				{name: "success then not found", ids: []string{good, missing}, status: http.StatusNotFound,
					wantItems: 1, wantRequests: []string{good, missing}, wantError: missing},
				{name: "not found then success", ids: []string{missing, good}, status: http.StatusNotFound,
					wantItems: 1, wantRequests: []string{missing, good}, wantError: missing},
				{name: "invalid then success", ids: []string{invalid, good}, wantItems: 1,
					wantRequests: []string{good}, wantError: invalidError},
				{name: "permission denied", ids: []string{missing}, status: http.StatusForbidden,
					wantRequests: []string{missing}, wantError: missing},
				{name: "timeout", ids: []string{missing}, wantError: missing, expired: true},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()

					for _, format := range []string{"json", "yaml", "short", "text"} {
						t.Run(format, func(t *testing.T) {
							t.Parallel()

							checkLookup(t, kind, format, tt)
						})
					}
				})
			}
		})
	}
}

const (
	good    = "00000000-0000-0000-0000-000000000001"
	missing = "00000000-0000-0000-0000-000000000002"
)

type lookupCase struct {
	name         string
	ids          []string
	status       int
	wantItems    int
	wantRequests []string
	wantError    string
	expired      bool
}

func checkLookup(t *testing.T, kind, format string, tt lookupCase) {
	t.Helper()

	var (
		mu       sync.Mutex
		requests []string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.HasPrefix(r.URL.Path, "/api/v1/auth/") {
			_, _ = io.WriteString(w, "{}")

			return
		}

		id := path.Base(r.URL.Path)

		mu.Lock()

		requests = append(requests, id)
		mu.Unlock()

		if id == missing {
			w.WriteHeader(tt.status)
			_, _ = io.WriteString(w, `{"detail":"lookup denied or missing"}`)

			return
		}

		_, _ = io.WriteString(w, `{"metadata":{"name":"`+good+`","namespace":"default","instanceUid":"`+good+`"},`+
			`"id":"`+good+`"}`)
	}))
	defer server.Close()

	conf := config.NewDefaultGlobalConfig(t.TempDir())
	conf.Clusters[0].OpAMPCommander.Endpoint = server.URL
	conf.Users[0].Auth.Type = config.AuthTypeManual
	conf.Users[0].Auth.BearerToken = "test-token"
	conf.Log.Logger = slog.New(slog.DiscardHandler)

	cmd := get.NewCommand(get.CommandOptions{GlobalConfig: conf})

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	outputFlag := "--output"
	if kind == "connection" {
		outputFlag = "--format"
	}

	cmd.SetArgs(append([]string{kind, outputFlag, format}, tt.ids...))

	ctx := t.Context()

	if tt.expired {
		var cancel context.CancelFunc

		ctx, cancel = context.WithDeadline(ctx, time.Now().Add(-time.Second))
		defer cancel()
	}

	err := cmd.ExecuteContext(ctx)
	if tt.wantError == "" {
		require.NoError(t, err)
		assert.Empty(t, stderr.String())
	} else {
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get "+kind)
		assert.Contains(t, stderr.String(), "failed to get "+kind)
		assert.Contains(t, stderr.String(), tt.wantError)

		if tt.status != 0 {
			var responseError *client.ResponseError
			require.ErrorAs(t, err, &responseError)
			assert.Equal(t, tt.status, responseError.StatusCode)
			assert.Contains(t, stderr.String(), "lookup denied or missing")
		}

		if tt.expired {
			require.ErrorIs(t, err, context.DeadlineExceeded)
		}
	}

	mu.Lock()
	assert.Equal(t, tt.wantRequests, requests)
	mu.Unlock()

	assert.NotContains(t, stdout.String(), "Usage:")
	assert.NotContains(t, stdout.String(), "No ")

	switch format {
	case "json":
		var items []json.RawMessage
		require.NoError(t, json.Unmarshal(stdout.Bytes(), &items))
		assert.Len(t, items, tt.wantItems)

		if tt.wantItems == 0 {
			assert.Equal(t, "[]", stdout.String())
		}
	case "yaml":
		var items []any
		require.NoError(t, yaml.Unmarshal(stdout.Bytes(), &items))
		assert.Len(t, items, tt.wantItems)
	case "short", "text":
		if tt.wantItems == 0 {
			assert.Empty(t, stdout.String())
		} else {
			assert.NotEmpty(t, stdout.String())
		}
	}
}

func TestGetFormatterFailure(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, "{}")
	}))
	defer server.Close()

	conf := config.NewDefaultGlobalConfig(t.TempDir())
	conf.Clusters[0].OpAMPCommander.Endpoint = server.URL
	conf.Users[0].Auth.Type = config.AuthTypeManual
	conf.Users[0].Auth.BearerToken = "test-token"
	conf.Log.Logger = slog.New(slog.DiscardHandler)

	cmd := get.NewCommand(get.CommandOptions{GlobalConfig: conf})
	cmd.SetOut(failingWriter{})
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"agent", "-o", "json", "invalid-uid", "00000000-0000-0000-0000-000000000001"})
	err := cmd.ExecuteContext(t.Context())
	require.ErrorContains(t, err, "invalid UID")
	require.ErrorIs(t, err, io.ErrClosedPipe)
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, io.ErrClosedPipe
}
