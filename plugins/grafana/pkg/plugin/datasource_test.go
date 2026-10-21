package plugin_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/minuk-dev/opampcommander/plugins/grafana/pkg/plugin"
)

func datasource(t *testing.T, endpoint string, allowHTTP bool) *plugin.Datasource {
	t.Helper()
	config, err := json.Marshal(map[string]any{"url": endpoint, "namespace": "default", "allowInsecureHTTP": allowHTTP})
	if err != nil {
		t.Fatal(err)
	}
	instance, err := plugin.NewDatasource(t.Context(), backend.DataSourceInstanceSettings{
		Type: "minukdev-opampcommander-datasource", JSONData: config, DecryptedSecureJSONData: map[string]string{"token": "private-token"},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(instance.(instancemgmt.InstanceDisposer).Dispose)
	return instance.(*plugin.Datasource)
}

func query(t *testing.T, ctx context.Context, ds *plugin.Datasource, body string) backend.DataResponse {
	t.Helper()
	response, err := ds.QueryData(ctx, &backend.QueryDataRequest{Queries: []backend.DataQuery{{RefID: "A", JSON: json.RawMessage(body)}}})
	if err != nil {
		t.Fatal(err)
	}
	return response.Responses["A"]
}

func TestFleetProjectionAndPagination(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/v1/namespaces/default/agents" || r.URL.Query().Get("limit") != "100" || r.URL.Query().Get("continue") != "next+page" || r.Header.Get("Authorization") != "Bearer private-token" {
			t.Error("incorrect bounded authenticated request")
		}
		fmt.Fprint(w, `{"metadata":{"continue":"last-page"},"items":[{"metadata":{"instanceUid":"agent-1","namespace":"default","capabilities":2048,"description":{"identifyingAttributes":{"service.instance.id":"collector-a","secret":"never-visible"}}},"spec":{"connectionSettings":{"password":"never-visible"}},"status":{"connected":true,"componentHealth":{"healthy":false},"effectiveConfig":{"secret":"never-visible"}}},{"metadata":{"instanceUid":"agent-2","namespace":"default"},"status":{"connected":false,"componentHealth":{"healthy":false}}}]}`)
	}))
	defer server.Close()
	result := query(t, t.Context(), datasource(t, server.URL, true), `{"queryType":"fleet","cursor":"next+page"}`)
	if result.Error != nil {
		t.Fatal(result.Error)
	}
	if len(result.Frames) != 1 || result.Frames[0].Rows() != 2 {
		t.Fatal("expected two fleet rows")
	}
	frame := result.Frames[0]
	if frame.Fields[1].At(0) != "collector-a" || frame.Fields[4].At(1) != "Disconnected" || frame.Fields[5].At(0) != "Unhealthy" || frame.Fields[5].At(1) != "Unknown" {
		t.Fatal("incorrect identity or status mapping")
	}
	if frame.Meta.Custom.(map[string]string)["nextCursor"] != "last-page" {
		t.Fatal("missing pagination cursor")
	}
	encoded, err := json.Marshal(frame)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "never-visible") || strings.Contains(string(encoded), "private-token") {
		t.Fatal("sensitive data escaped into frame")
	}
}

func TestFleetFailsClosed(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		status int
		body   string
		want   backend.Status
	}{
		{"expired token", 401, "private-token", backend.StatusUnauthorized},
		{"forbidden namespace", 403, "private-token", backend.StatusForbidden},
		{"server failure", 500, "private-token", backend.StatusBadGateway},
		{"redirect", 302, "private-token", backend.StatusBadGateway},
		{"malformed", 200, `{}`, backend.StatusBadGateway},
		{"wrong namespace", 200, `{"items":[{"metadata":{"instanceUid":"x","namespace":"other"}}]}`, backend.StatusBadGateway},
		{"oversized", 200, strings.Repeat("x", (4<<20)+1), backend.StatusBadGateway},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/redirect-target" {
					t.Error("followed credential-bearing redirect")
				}
				w.Header().Set("Location", "/redirect-target")
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			result := query(t, t.Context(), datasource(t, server.URL, true), `{}`)
			if result.Error == nil || result.Status != tc.want || len(result.Frames) != 0 {
				t.Fatalf("expected error %v without frames, got %+v", tc.want, result)
			}
			if strings.Contains(result.Error.Error(), "private-token") {
				t.Fatal("upstream body exposed")
			}
		})
	}
}

func TestSettingsAndQueryValidation(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, endpoint, body string
		allowHTTP            bool
	}{
		{"unconfigured", "", `{}`, false},
		{"insecure", "http://localhost:1", `{}`, false},
		{"url credentials", "https://user:password@localhost", `{}`, false},
		{"url query", "https://localhost?token=secret", `{}`, false},
		{"query type", "https://localhost", `{"queryType":"write"}`, false},
		{"invalid json", "https://localhost", `{`, false},
		{"large cursor", "https://localhost", `{"cursor":"` + strings.Repeat("x", 4097) + `"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := query(t, t.Context(), datasource(t, tc.endpoint, tc.allowHTTP), tc.body)
			if result.Error == nil || result.Status != backend.StatusBadRequest {
				t.Fatalf("expected validation failure, got %+v", result)
			}
		})
	}
}

func TestHealthChecksProtectedResourceAndEmptyFleet(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/namespaces/default/agents" || r.Header.Get("Authorization") != "Bearer private-token" {
			t.Error("health did not check authorized agent read")
		}
		fmt.Fprint(w, `{"items":[]}`)
	}))
	defer server.Close()
	ds := datasource(t, server.URL, true)
	health, err := ds.CheckHealth(t.Context(), &backend.CheckHealthRequest{})
	if err != nil || health.Status != backend.HealthStatusOk {
		t.Fatalf("health: %+v, %v", health, err)
	}
	result := query(t, t.Context(), ds, `{}`)
	if result.Error != nil || len(result.Frames) != 1 || result.Frames[0].Rows() != 0 {
		t.Fatalf("empty fleet is not a valid empty frame: %+v", result)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if result := query(t, ctx, ds, `{}`); result.Error == nil {
		t.Fatal("cancelled request succeeded")
	}
}
