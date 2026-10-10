package primary_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/internal/module/adapter/primary"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/management/observability"
)

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
