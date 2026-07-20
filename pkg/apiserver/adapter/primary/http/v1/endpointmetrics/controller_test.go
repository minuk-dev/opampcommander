package endpointmetrics_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"go.uber.org/goleak"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/endpointmetrics"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	goleak.VerifyTestMain(m)
}

var errBoom = errors.New("boom")

// mockUsecase is a testify mock of usecase.EndpointMetricsUsecase.
type mockUsecase struct {
	mock.Mock
}

func newMockUsecase(t *testing.T) *mockUsecase {
	t.Helper()

	m := &mockUsecase{}
	m.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}

func (m *mockUsecase) GetEndpointThroughput(
	ctx context.Context, namespace, name string, window time.Duration,
) (*v1.EndpointThroughput, error) {
	args := m.Called(ctx, namespace, name, window)

	res, _ := args.Get(0).(*v1.EndpointThroughput)

	return res, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockUsecase) ListEndpointThroughput(
	ctx context.Context, namespace string, window time.Duration,
) (*v1.ListResponse[v1.EndpointThroughput], error) {
	args := m.Called(ctx, namespace, window)

	res, _ := args.Get(0).(*v1.ListResponse[v1.EndpointThroughput])

	return res, args.Error(1) //nolint:wrapcheck // mock error
}

func setup(t *testing.T) (*testutil.ControllerBase, *mockUsecase) {
	t.Helper()

	ctrlBase := testutil.NewBase(t).ForController()
	usecase := newMockUsecase(t)
	controller := endpointmetrics.NewController(usecase, slog.Default())
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

func TestEndpointMetricsController_RoutesInfo(t *testing.T) {
	t.Parallel()

	controller := endpointmetrics.NewController(newMockUsecase(t), slog.Default())

	routes := controller.RoutesInfo()
	require.Len(t, routes, 2)

	got := make(map[string]struct{}, len(routes))
	for _, route := range routes {
		got[route.Method+" "+route.Path] = struct{}{}

		assert.NotNil(t, route.HandlerFunc)
	}

	assert.Contains(t, got, "GET /api/v1/namespaces/:namespace/endpoint-throughputs")
	assert.Contains(t, got, "GET /api/v1/namespaces/:namespace/endpoints/:name/throughput")
}

func TestEndpointMetricsController_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns the endpoint throughput", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		//exhaustruct:ignore
		usecase.On("GetEndpointThroughput", mock.Anything, "default", "otlp", 5*time.Minute).
			Return(&v1.EndpointThroughput{Namespace: "default", Name: "otlp"}, nil)

		recorder := doGET(t, ctrlBase.Router,
			"/api/v1/namespaces/default/endpoints/otlp/throughput?window=5m")

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "otlp", gjson.Get(recorder.Body.String(), "name").String())
	})

	t.Run("returns 400 on an invalid window", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := setup(t)

		recorder := doGET(t, ctrlBase.Router,
			"/api/v1/namespaces/default/endpoints/otlp/throughput?window=nope")

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("returns 404 when the endpoint does not exist", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("GetEndpointThroughput", mock.Anything, "default", "missing", time.Duration(0)).
			Return(nil, model.ErrResourceNotExist)

		recorder := doGET(t, ctrlBase.Router,
			"/api/v1/namespaces/default/endpoints/missing/throughput")

		require.Equal(t, http.StatusNotFound, recorder.Code)
	})
}

func TestEndpointMetricsController_List(t *testing.T) {
	t.Parallel()

	t.Run("returns the throughput list", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		//exhaustruct:ignore
		usecase.On("ListEndpointThroughput", mock.Anything, "default", time.Duration(0)).
			Return(&v1.ListResponse[v1.EndpointThroughput]{
				Items: []v1.EndpointThroughput{{Namespace: "default", Name: "otlp"}},
			}, nil)

		recorder := doGET(t, ctrlBase.Router, "/api/v1/namespaces/default/endpoint-throughputs")

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, int64(1), gjson.Get(recorder.Body.String(), "items.#").Int())
	})

	t.Run("returns 400 on an invalid window", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := setup(t)

		recorder := doGET(t, ctrlBase.Router, "/api/v1/namespaces/default/endpoint-throughputs?window=nope")

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("returns 500 when the usecase fails", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("ListEndpointThroughput", mock.Anything, "default", time.Duration(0)).Return(nil, errBoom)

		recorder := doGET(t, ctrlBase.Router, "/api/v1/namespaces/default/endpoint-throughputs")

		require.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}

// TestEndpointMetricsController_MissingParams covers the required-path-param validation
// branches, unreachable through routing since the segments are never empty when matched.
func TestEndpointMetricsController_MissingParams(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		pick   func(*endpointmetrics.Controller) gin.HandlerFunc
		params gin.Params
	}{
		{"List missing namespace", func(c *endpointmetrics.Controller) gin.HandlerFunc { return c.List }, gin.Params{}},
		{"Get missing namespace", func(c *endpointmetrics.Controller) gin.HandlerFunc { return c.Get }, gin.Params{}},
		{
			"Get missing name",
			func(c *endpointmetrics.Controller) gin.HandlerFunc { return c.Get },
			gin.Params{{Key: "namespace", Value: "default"}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			controller := endpointmetrics.NewController(newMockUsecase(t), slog.Default())

			recorder := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(recorder)
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
			require.NoError(t, err)

			ginCtx.Request = req
			ginCtx.Params = tc.params

			tc.pick(controller)(ginCtx)

			require.Equal(t, http.StatusBadRequest, recorder.Code)
		})
	}
}
