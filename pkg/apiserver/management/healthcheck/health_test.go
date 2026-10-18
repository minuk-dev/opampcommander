package healthcheck_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/management/healthcheck"
)

func TestHealthHelper_BeginShutdown(t *testing.T) {
	t.Parallel()

	helper := healthcheck.NewHealthHelper(nil)
	ready, _ := helper.Readiness(t.Context())
	assert.True(t, ready)
	helper.BeginShutdown()

	recorder := httptest.NewRecorder()
	helper.IsReady(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/readyz", nil))
	assert.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "server is shutting down")
	healthy, _ := helper.Health(t.Context())
	assert.True(t, healthy)
}
