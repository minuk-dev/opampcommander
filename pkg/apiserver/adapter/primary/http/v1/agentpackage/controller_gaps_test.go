package agentpackage_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/agentpackage"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/agentpackage/usecasemock"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func gapRouter(t *testing.T) *gin.Engine {
	t.Helper()

	ctrlBase := testutil.NewBase(t).ForController()
	controller := agentpackage.NewController(usecasemock.NewMockUsecase(t), ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)

	return ctrlBase.Router
}

func gapGET(t *testing.T, router *gin.Engine, target string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)

	return recorder
}

const gapBase = "/api/v1/namespaces/default/agentpackages"

func TestAgentPackageController_ListValidation(t *testing.T) {
	t.Parallel()

	t.Run("invalid limit", func(t *testing.T) {
		t.Parallel()

		recorder := gapGET(t, gapRouter(t), gapBase+"?limit=abc")

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("invalid includeDeleted", func(t *testing.T) {
		t.Parallel()

		recorder := gapGET(t, gapRouter(t), gapBase+"?includeDeleted=maybe")

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})
}

func TestAgentPackageController_GetValidation(t *testing.T) {
	t.Parallel()

	recorder := gapGET(t, gapRouter(t), gapBase+"/pkg-1?includeDeleted=nope")

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

// TestAgentPackageController_MissingParams covers the required :namespace / :name validation
// branches across every handler, unreachable through routing since the segments are never empty.
func TestAgentPackageController_MissingParams(t *testing.T) {
	t.Parallel()

	controller := agentpackage.NewController(usecasemock.NewMockUsecase(t), testutil.NewBase(t).Logger)
	ns := gin.Params{{Key: "namespace", Value: "default"}}

	cases := []struct {
		name    string
		handler gin.HandlerFunc
		params  gin.Params
	}{
		{"Get missing namespace", controller.Get, nil},
		{"Get missing name", controller.Get, ns},
		{"Create missing namespace", controller.Create, nil},
		{"Update missing namespace", controller.Update, nil},
		{"Update missing name", controller.Update, ns},
		{"Delete missing namespace", controller.Delete, nil},
		{"Delete missing name", controller.Delete, ns},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			recorder := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(recorder)
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
			require.NoError(t, err)

			ginCtx.Request = req
			ginCtx.Params = tc.params

			tc.handler(ginCtx)

			require.Equal(t, http.StatusBadRequest, recorder.Code)
		})
	}
}
