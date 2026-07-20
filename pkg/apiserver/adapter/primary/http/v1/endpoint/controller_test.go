package endpoint_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"go.uber.org/goleak"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/endpoint"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	goleak.VerifyTestMain(m)
}

var errBoom = errors.New("boom")

// mockUsecase is a testify mock of usecase.EndpointManageUsecase.
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

func (m *mockUsecase) GetEndpoint(
	ctx context.Context, namespace, name string, options *port.GetOptions,
) (*v1.Endpoint, error) {
	args := m.Called(ctx, namespace, name, options)

	res, _ := args.Get(0).(*v1.Endpoint)

	return res, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockUsecase) ListEndpoints(
	ctx context.Context, namespace string, options *port.ListOptions,
) (*v1.ListResponse[v1.Endpoint], error) {
	args := m.Called(ctx, namespace, options)

	res, _ := args.Get(0).(*v1.ListResponse[v1.Endpoint])

	return res, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockUsecase) CreateEndpoint(ctx context.Context, e *v1.Endpoint) (*v1.Endpoint, error) {
	args := m.Called(ctx, e)

	res, _ := args.Get(0).(*v1.Endpoint)

	return res, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockUsecase) UpdateEndpoint(
	ctx context.Context, namespace, name string, e *v1.Endpoint,
) (*v1.Endpoint, error) {
	args := m.Called(ctx, namespace, name, e)

	res, _ := args.Get(0).(*v1.Endpoint)

	return res, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockUsecase) DeleteEndpoint(ctx context.Context, namespace, name string) error {
	args := m.Called(ctx, namespace, name)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func newEndpoint() *v1.Endpoint {
	//exhaustruct:ignore
	return &v1.Endpoint{
		Kind:       v1.EndpointKind,
		APIVersion: v1.APIVersion,
		//exhaustruct:ignore
		Metadata: v1.EndpointMetadata{Name: "otlp", Namespace: "default"},
	}
}

func setup(t *testing.T) (*testutil.ControllerBase, *mockUsecase) {
	t.Helper()

	ctrlBase := testutil.NewBase(t).ForController()
	usecase := newMockUsecase(t)
	controller := endpoint.NewController(usecase, slog.Default())
	ctrlBase.SetupRouter(controller)

	return ctrlBase, usecase
}

func doReq(t *testing.T, router *gin.Engine, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), method, target, strings.NewReader(body))
	require.NoError(t, err)

	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	router.ServeHTTP(recorder, req)

	return recorder
}

const base = "/api/v1/namespaces/default/endpoints"

func TestEndpointController_RoutesInfo(t *testing.T) {
	t.Parallel()

	controller := endpoint.NewController(newMockUsecase(t), slog.Default())

	routes := controller.RoutesInfo()
	require.Len(t, routes, 5)

	got := make(map[string]struct{}, len(routes))
	for _, route := range routes {
		got[route.Method+" "+route.Path] = struct{}{}

		assert.NotNil(t, route.HandlerFunc)
	}

	for _, want := range []string{
		"GET /api/v1/namespaces/:namespace/endpoints",
		"GET /api/v1/namespaces/:namespace/endpoints/:name",
		"POST /api/v1/namespaces/:namespace/endpoints",
		"PUT /api/v1/namespaces/:namespace/endpoints/:name",
		"DELETE /api/v1/namespaces/:namespace/endpoints/:name",
	} {
		assert.Contains(t, got, want)
	}
}

func TestEndpointController_List(t *testing.T) {
	t.Parallel()

	t.Run("returns the list of endpoints", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		//exhaustruct:ignore
		usecase.On("ListEndpoints", mock.Anything, "default", mock.Anything).Return(&v1.ListResponse[v1.Endpoint]{
			Items: []v1.Endpoint{*newEndpoint()},
		}, nil)

		recorder := doReq(t, ctrlBase.Router, http.MethodGet, base+"?limit=10&includeDeleted=true", "")

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, int64(1), gjson.Get(recorder.Body.String(), "items.#").Int())
	})

	t.Run("returns 400 on an invalid limit", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := setup(t)

		recorder := doReq(t, ctrlBase.Router, http.MethodGet, base+"?limit=abc", "")

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("returns 400 on an invalid includeDeleted", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := setup(t)

		recorder := doReq(t, ctrlBase.Router, http.MethodGet, base+"?includeDeleted=maybe", "")

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("returns 500 when the usecase fails", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("ListEndpoints", mock.Anything, "default", mock.Anything).Return(nil, errBoom)

		recorder := doReq(t, ctrlBase.Router, http.MethodGet, base, "")

		require.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}

func TestEndpointController_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns the endpoint", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("GetEndpoint", mock.Anything, "default", "otlp", mock.Anything).
			Return(newEndpoint(), nil)

		recorder := doReq(t, ctrlBase.Router, http.MethodGet, base+"/otlp", "")

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "otlp", gjson.Get(recorder.Body.String(), "metadata.name").String())
	})

	t.Run("returns 400 on an invalid includeDeleted", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := setup(t)

		recorder := doReq(t, ctrlBase.Router, http.MethodGet, base+"/otlp?includeDeleted=nope", "")

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("returns 404 when the endpoint does not exist", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("GetEndpoint", mock.Anything, "default", "missing", mock.Anything).
			Return(nil, model.ErrResourceNotExist)

		recorder := doReq(t, ctrlBase.Router, http.MethodGet, base+"/missing", "")

		require.Equal(t, http.StatusNotFound, recorder.Code)
	})
}

func TestEndpointController_Create(t *testing.T) {
	t.Parallel()

	t.Run("creates an endpoint and sets the Location header", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("CreateEndpoint", mock.Anything, mock.Anything).Return(newEndpoint(), nil)

		recorder := doReq(t, ctrlBase.Router, http.MethodPost, base, `{"metadata":{"name":"otlp"}}`)

		require.Equal(t, http.StatusCreated, recorder.Code)
		assert.Equal(t, base+"/otlp", recorder.Header().Get("Location"))
	})

	t.Run("returns 400 on an invalid body", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := setup(t)

		recorder := doReq(t, ctrlBase.Router, http.MethodPost, base, `{not-json`)

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("returns 409 when the endpoint already exists", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("CreateEndpoint", mock.Anything, mock.Anything).Return(nil, model.ErrResourceAlreadyExist)

		recorder := doReq(t, ctrlBase.Router, http.MethodPost, base, `{"metadata":{"name":"otlp"}}`)

		require.Equal(t, http.StatusConflict, recorder.Code)
	})
}

func TestEndpointController_Update(t *testing.T) {
	t.Parallel()

	t.Run("updates an endpoint", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("UpdateEndpoint", mock.Anything, "default", "otlp", mock.Anything).
			Return(newEndpoint(), nil)

		recorder := doReq(t, ctrlBase.Router, http.MethodPut, base+"/otlp", `{"metadata":{"name":"otlp"}}`)

		require.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("returns 400 on an invalid body", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := setup(t)

		recorder := doReq(t, ctrlBase.Router, http.MethodPut, base+"/otlp", `{not-json`)

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("returns 404 when the endpoint does not exist", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("UpdateEndpoint", mock.Anything, "default", "missing", mock.Anything).
			Return(nil, model.ErrResourceNotExist)

		recorder := doReq(t, ctrlBase.Router, http.MethodPut, base+"/missing", `{"metadata":{"name":"missing"}}`)

		require.Equal(t, http.StatusNotFound, recorder.Code)
	})
}

func TestEndpointController_Delete(t *testing.T) {
	t.Parallel()

	t.Run("deletes an endpoint", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("DeleteEndpoint", mock.Anything, "default", "otlp").Return(nil)

		recorder := doReq(t, ctrlBase.Router, http.MethodDelete, base+"/otlp", "")

		require.Equal(t, http.StatusNoContent, recorder.Code)
	})

	t.Run("returns 404 when the endpoint does not exist", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("DeleteEndpoint", mock.Anything, "default", "missing").Return(model.ErrResourceNotExist)

		recorder := doReq(t, ctrlBase.Router, http.MethodDelete, base+"/missing", "")

		require.Equal(t, http.StatusNotFound, recorder.Code)
	})
}

// TestEndpointController_MissingParams covers the required :namespace / :name validation branches
// across every handler, unreachable through routing since the segments are never empty when matched.
func TestEndpointController_MissingParams(t *testing.T) {
	t.Parallel()

	// No usecase method is reached (every handler returns at the validation guard), so a single
	// controller with an expectation-free mock is shared across all cases.
	controller := endpoint.NewController(newMockUsecase(t), slog.Default())
	ns := gin.Params{{Key: "namespace", Value: "default"}}

	cases := []struct {
		name    string
		handler gin.HandlerFunc
		params  gin.Params
	}{
		{"List missing namespace", controller.List, nil},
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
