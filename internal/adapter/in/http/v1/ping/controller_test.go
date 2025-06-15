package ping_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/ping"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestPingController_Handle(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)
	ctrlBase := base.ForController()

	controller := ping.NewController(base.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/ping", nil)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, "{\"message\":\"pong\"}", recorder.Body.String())
}
