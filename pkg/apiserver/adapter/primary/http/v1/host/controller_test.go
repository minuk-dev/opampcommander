package host_test

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"go.uber.org/goleak"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/host"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/host/usecasemock"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	goleak.VerifyTestMain(m)
}

var errBoom = errors.New("boom")

func setup(t *testing.T) (*testutil.ControllerBase, *usecasemock.MockManageUsecase) {
	t.Helper()

	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockManageUsecase(t)
	controller := host.NewController(usecase, slog.Default())
	ctrlBase.SetupRouter(controller)

	return ctrlBase, usecase
}

func doGET(t *testing.T, router *gin.Engine, target string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)

	return recorder
}

func TestHostController_RoutesInfo(t *testing.T) {
	t.Parallel()

	controller := host.NewController(usecasemock.NewMockManageUsecase(t), slog.Default())

	routes := controller.RoutesInfo()
	require.Len(t, routes, 3)

	got := make(map[string]struct{}, len(routes))
	for _, route := range routes {
		got[route.Method+" "+route.Path] = struct{}{}

		assert.NotNil(t, route.HandlerFunc)
	}

	for _, want := range []string{
		"GET /api/v1/hosts",
		"GET /api/v1/hosts/:id",
		"GET /api/v1/hosts/:id/agents",
	} {
		assert.Contains(t, got, want)
	}
}

func TestHostController_List(t *testing.T) {
	t.Parallel()

	t.Run("returns the list of hosts", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		//exhaustruct:ignore
		usecase.On("ListHosts", mock.Anything, mock.Anything).Return(&v1.ListResponse[v1.Host]{
			Items: []v1.Host{{Kind: v1.HostKind}},
		}, nil)

		recorder := doGET(t, ctrlBase.Router, "/api/v1/hosts?limit=10")

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, int64(1), gjson.Get(recorder.Body.String(), "items.#").Int())
	})

	t.Run("returns 400 on an invalid limit", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := setup(t)

		recorder := doGET(t, ctrlBase.Router, "/api/v1/hosts?limit=abc")

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("returns 500 when the usecase fails", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("ListHosts", mock.Anything, mock.Anything).Return(nil, errBoom)

		recorder := doGET(t, ctrlBase.Router, "/api/v1/hosts")

		require.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}

func TestHostController_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns the host", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("GetHost", mock.Anything, "h-1").Return(&v1.Host{Kind: v1.HostKind}, nil)

		recorder := doGET(t, ctrlBase.Router, "/api/v1/hosts/h-1")

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, v1.HostKind, gjson.Get(recorder.Body.String(), "kind").String())
	})

	t.Run("returns 404 when the host does not exist", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("GetHost", mock.Anything, "missing").Return(nil, model.ErrResourceNotExist)

		recorder := doGET(t, ctrlBase.Router, "/api/v1/hosts/missing")

		require.Equal(t, http.StatusNotFound, recorder.Code)
	})
}

func TestHostController_ListAgents(t *testing.T) {
	t.Parallel()

	t.Run("returns the host agents", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		//exhaustruct:ignore
		usecase.On("ListAgentsByHost", mock.Anything, "h-1", mock.Anything).
			Return(&v1.ListResponse[v1.Agent]{Items: []v1.Agent{}}, nil)

		recorder := doGET(t, ctrlBase.Router, "/api/v1/hosts/h-1/agents")

		require.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("returns 400 on an invalid limit", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := setup(t)

		recorder := doGET(t, ctrlBase.Router, "/api/v1/hosts/h-1/agents?limit=abc")

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("returns 404 when the host does not exist", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("ListAgentsByHost", mock.Anything, "missing", mock.Anything).
			Return(nil, model.ErrResourceNotExist)

		recorder := doGET(t, ctrlBase.Router, "/api/v1/hosts/missing/agents")

		require.Equal(t, http.StatusNotFound, recorder.Code)
	})
}

// TestHostController_MissingID covers the required-:id validation branches in Get and
// ListAgents, unreachable through routing since the path segment is never empty when matched.
func TestHostController_MissingID(t *testing.T) {
	t.Parallel()

	handlers := map[string]func(*host.Controller) gin.HandlerFunc{
		"Get":        func(c *host.Controller) gin.HandlerFunc { return c.Get },
		"ListAgents": func(c *host.Controller) gin.HandlerFunc { return c.ListAgents },
	}

	for name, pick := range handlers {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			controller := host.NewController(usecasemock.NewMockManageUsecase(t), slog.Default())

			recorder := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(recorder)
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
			require.NoError(t, err)

			ginCtx.Request = req

			pick(controller)(ginCtx)

			require.Equal(t, http.StatusBadRequest, recorder.Code)
		})
	}
}
