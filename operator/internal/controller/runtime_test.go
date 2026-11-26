package controller

import (
	"context"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestRuntimeNamespaceVerification(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, body string
		code       int
		ok         bool
	}{
		{"match", `{"name":"tenant","installationUID":"anchor","namespaceUID":"tenant-id"}`, 200, true},
		{"recreated namespace", `{"name":"tenant","installationUID":"anchor","namespaceUID":"other"}`, 200, false},
		{"other installation", `{"name":"tenant","installationUID":"other","namespaceUID":"tenant-id"}`, 200, false},
		{"trailing data", `{"name":"tenant","installationUID":"anchor","namespaceUID":"tenant-id"}{}`, 200, false},
		{"denied", "", 403, false}, {"redirect", "", 302, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/runtime/v1/namespaces/tenant" || r.Header.Get("Authorization") != "" {
					t.Error("incorrect scoped request")
				}
				if tc.code == 302 {
					w.Header().Set("Location", "/other")
				}
				w.WriteHeader(tc.code)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			dir := t.TempDir()
			ca := filepath.Join(dir, "ca")
			if err := os.WriteFile(ca, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}), 0600); err != nil {
				t.Fatal(err)
			}
			c, err := NewRuntimeClient(server.URL, ca)
			if err != nil {
				t.Fatal(err)
			}
			err = c.verify(t.Context(), "tenant", "anchor", "tenant-id")
			if (err == nil) != tc.ok {
				t.Fatalf("unexpected verification result: %v", err)
			}
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			if err := c.verify(ctx, "tenant", "anchor", "tenant-id"); err == nil {
				t.Error("ignored cancellation")
			}
		})
	}
}

func TestRuntimeClientRequiresHTTPSOrigin(t *testing.T) {
	t.Parallel()
	for _, address := range []string{"", "http://commander", "https://user:password@commander", "https://commander/path", "https://commander?token=x", "https://commander#fragment"} {
		if _, err := NewRuntimeClient(address, ""); err == nil {
			t.Errorf("accepted invalid runtime origin %q", address)
		}
	}

}
