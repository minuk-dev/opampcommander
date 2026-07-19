package version_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"go.uber.org/goleak"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/version"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestVersionController_RoutesInfo(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)
	controller := version.NewController(base.Logger)

	routes := controller.RoutesInfo()
	require.Len(t, routes, 1)
	assert.Equal(t, http.MethodGet, routes[0].Method)
	assert.Equal(t, "/api/v1/version", routes[0].Path)
	assert.NotNil(t, routes[0].HandlerFunc)
}

func TestVersionController_GetVersion(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)
	ctrlBase := base.ForController()

	controller := version.NewController(base.Logger)
	ctrlBase.SetupRouter(controller)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/version", nil)
	require.NoError(t, err)
	ctrlBase.Router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	// goVersion is always populated from the runtime, so it is a stable field to assert on.
	assert.NotEmpty(t, gjson.Get(recorder.Body.String(), "goVersion").String())
	assert.NotEmpty(t, gjson.Get(recorder.Body.String(), "platform").String())
}
