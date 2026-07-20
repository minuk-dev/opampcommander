package container_test

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
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/container"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/container/usecasemock"
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
	controller := container.NewController(usecase, slog.Default())
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

func TestContainerController_RoutesInfo(t *testing.T) {
	t.Parallel()

	controller := container.NewController(usecasemock.NewMockManageUsecase(t), slog.Default())

	routes := controller.RoutesInfo()
	require.Len(t, routes, 3)

	got := make(map[string]struct{}, len(routes))
	for _, route := range routes {
		got[route.Method+" "+route.Path] = struct{}{}

		assert.NotNil(t, route.HandlerFunc)
	}

	for _, want := range []string{
		"GET /api/v1/containers",
		"GET /api/v1/containers/:id",
		"GET /api/v1/containers/:id/agents",
	} {
		assert.Contains(t, got, want)
	}
}

func TestContainerController_List(t *testing.T) {
	t.Parallel()

	t.Run("returns the list of containers", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		//exhaustruct:ignore
		usecase.On("ListContainers", mock.Anything, mock.Anything).Return(&v1.ListResponse[v1.Container]{
			Items: []v1.Container{{Kind: v1.ContainerKind}},
		}, nil)

		recorder := doGET(t, ctrlBase.Router, "/api/v1/containers?limit=10")

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, int64(1), gjson.Get(recorder.Body.String(), "items.#").Int())
	})

	t.Run("returns 400 on an invalid limit", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := setup(t)

		recorder := doGET(t, ctrlBase.Router, "/api/v1/containers?limit=abc")

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("returns 500 when the usecase fails", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("ListContainers", mock.Anything, mock.Anything).Return(nil, errBoom)

		recorder := doGET(t, ctrlBase.Router, "/api/v1/containers")

		require.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}

func TestContainerController_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns the container", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("GetContainer", mock.Anything, "c-1").Return(&v1.Container{Kind: v1.ContainerKind}, nil)

		recorder := doGET(t, ctrlBase.Router, "/api/v1/containers/c-1")

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, v1.ContainerKind, gjson.Get(recorder.Body.String(), "kind").String())
	})

	t.Run("returns 404 when the container does not exist", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("GetContainer", mock.Anything, "missing").Return(nil, model.ErrResourceNotExist)

		recorder := doGET(t, ctrlBase.Router, "/api/v1/containers/missing")

		require.Equal(t, http.StatusNotFound, recorder.Code)
	})
}

func TestContainerController_ListAgents(t *testing.T) {
	t.Parallel()

	t.Run("returns the container agents", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		//exhaustruct:ignore
		usecase.On("ListAgentsByContainer", mock.Anything, "c-1", mock.Anything).
			Return(&v1.ListResponse[v1.Agent]{Items: []v1.Agent{}}, nil)

		recorder := doGET(t, ctrlBase.Router, "/api/v1/containers/c-1/agents")

		require.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("returns 400 on an invalid limit", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := setup(t)

		recorder := doGET(t, ctrlBase.Router, "/api/v1/containers/c-1/agents?limit=abc")

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("returns 404 when the container does not exist", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("ListAgentsByContainer", mock.Anything, "missing", mock.Anything).
			Return(nil, model.ErrResourceNotExist)

		recorder := doGET(t, ctrlBase.Router, "/api/v1/containers/missing/agents")

		require.Equal(t, http.StatusNotFound, recorder.Code)
	})
}

// TestContainerController_MissingID covers the required-:id validation branches in Get and
// ListAgents, unreachable through routing since the path segment is never empty when matched.
func TestContainerController_MissingID(t *testing.T) {
	t.Parallel()

	handlers := map[string]func(*container.Controller) gin.HandlerFunc{
		"Get":        func(c *container.Controller) gin.HandlerFunc { return c.Get },
		"ListAgents": func(c *container.Controller) gin.HandlerFunc { return c.ListAgents },
	}

	for name, pick := range handlers {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			controller := container.NewController(usecasemock.NewMockManageUsecase(t), slog.Default())

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
