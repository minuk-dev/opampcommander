package namespace_test

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
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/namespace"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	goleak.VerifyTestMain(m)
}

var errBoom = errors.New("boom")

// mockNamespaceUsecase is a testify mock of usecase.NamespaceManageUsecase.
type mockNamespaceUsecase struct {
	mock.Mock
}

func newMockNamespaceUsecase(t *testing.T) *mockNamespaceUsecase {
	t.Helper()

	m := &mockNamespaceUsecase{}
	m.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}

func (m *mockNamespaceUsecase) GetNamespace(
	ctx context.Context, name string, options *port.GetOptions,
) (*v1.Namespace, error) {
	args := m.Called(ctx, name, options)

	res, _ := args.Get(0).(*v1.Namespace)

	return res, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockNamespaceUsecase) ListNamespaces(
	ctx context.Context, options *port.ListOptions,
) (*v1.ListResponse[v1.Namespace], error) {
	args := m.Called(ctx, options)

	res, _ := args.Get(0).(*v1.ListResponse[v1.Namespace])

	return res, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockNamespaceUsecase) CreateNamespace(
	ctx context.Context, ns *v1.Namespace,
) (*v1.Namespace, error) {
	args := m.Called(ctx, ns)

	res, _ := args.Get(0).(*v1.Namespace)

	return res, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockNamespaceUsecase) UpdateNamespace(
	ctx context.Context, name string, ns *v1.Namespace,
) (*v1.Namespace, error) {
	args := m.Called(ctx, name, ns)

	res, _ := args.Get(0).(*v1.Namespace)

	return res, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockNamespaceUsecase) DeleteNamespace(ctx context.Context, name string) error {
	args := m.Called(ctx, name)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func newNamespace(name string) *v1.Namespace {
	//exhaustruct:ignore
	return &v1.Namespace{
		Kind:       v1.NamespaceKind,
		APIVersion: v1.APIVersion,
		//exhaustruct:ignore
		Metadata: v1.NamespaceMetadata{Name: name},
	}
}

func setup(t *testing.T) (*testutil.ControllerBase, *mockNamespaceUsecase) {
	t.Helper()

	ctrlBase := testutil.NewBase(t).ForController()
	usecase := newMockNamespaceUsecase(t)
	controller := namespace.NewController(usecase, slog.Default())
	ctrlBase.SetupRouter(controller)

	return ctrlBase, usecase
}

func doReq(t *testing.T, router *gin.Engine, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()

	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
	} else {
		reader = strings.NewReader("")
	}

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), method, target, reader)
	require.NoError(t, err)

	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	router.ServeHTTP(recorder, req)

	return recorder
}

func TestNamespaceController_RoutesInfo(t *testing.T) {
	t.Parallel()

	controller := namespace.NewController(newMockNamespaceUsecase(t), slog.Default())

	routes := controller.RoutesInfo()
	require.Len(t, routes, 5)

	got := make(map[string]struct{}, len(routes))
	for _, route := range routes {
		got[route.Method+" "+route.Path] = struct{}{}

		assert.NotNil(t, route.HandlerFunc)
	}

	for _, want := range []string{
		"GET /api/v1/namespaces",
		"GET /api/v1/namespaces/:namespace",
		"POST /api/v1/namespaces",
		"PUT /api/v1/namespaces/:namespace",
		"DELETE /api/v1/namespaces/:namespace",
	} {
		assert.Contains(t, got, want)
	}
}

func TestNamespaceController_List(t *testing.T) {
	t.Parallel()

	t.Run("returns the list of namespaces", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		//exhaustruct:ignore
		usecase.On("ListNamespaces", mock.Anything, mock.Anything).Return(&v1.ListResponse[v1.Namespace]{
			Items: []v1.Namespace{*newNamespace("default"), *newNamespace("prod")},
		}, nil)

		recorder := doReq(t, ctrlBase.Router, http.MethodGet, "/api/v1/namespaces?limit=10&includeDeleted=true", "")

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, int64(2), gjson.Get(recorder.Body.String(), "items.#").Int())
	})

	t.Run("returns 400 on an invalid limit", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := setup(t)

		recorder := doReq(t, ctrlBase.Router, http.MethodGet, "/api/v1/namespaces?limit=abc", "")

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("returns 400 on an invalid includeDeleted", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := setup(t)

		recorder := doReq(t, ctrlBase.Router, http.MethodGet, "/api/v1/namespaces?includeDeleted=maybe", "")

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("returns 500 when the usecase fails", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("ListNamespaces", mock.Anything, mock.Anything).Return(nil, errBoom)

		recorder := doReq(t, ctrlBase.Router, http.MethodGet, "/api/v1/namespaces", "")

		require.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}

func TestNamespaceController_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns the namespace", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("GetNamespace", mock.Anything, "default", mock.Anything).Return(newNamespace("default"), nil)

		recorder := doReq(t, ctrlBase.Router, http.MethodGet, "/api/v1/namespaces/default", "")

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "default", gjson.Get(recorder.Body.String(), "metadata.name").String())
	})

	t.Run("returns 400 on an invalid includeDeleted", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := setup(t)

		recorder := doReq(t, ctrlBase.Router, http.MethodGet, "/api/v1/namespaces/default?includeDeleted=nope", "")

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("returns 404 when the namespace does not exist", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("GetNamespace", mock.Anything, "missing", mock.Anything).Return(nil, model.ErrResourceNotExist)

		recorder := doReq(t, ctrlBase.Router, http.MethodGet, "/api/v1/namespaces/missing", "")

		require.Equal(t, http.StatusNotFound, recorder.Code)
	})
}

func TestNamespaceController_Create(t *testing.T) {
	t.Parallel()

	t.Run("creates a namespace and sets the Location header", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("CreateNamespace", mock.Anything, mock.Anything).Return(newNamespace("prod"), nil)

		recorder := doReq(t, ctrlBase.Router, http.MethodPost, "/api/v1/namespaces", `{"metadata":{"name":"prod"}}`)

		require.Equal(t, http.StatusCreated, recorder.Code)
		assert.Equal(t, "/api/v1/namespaces/prod", recorder.Header().Get("Location"))
	})

	t.Run("returns 400 on an invalid body", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := setup(t)

		recorder := doReq(t, ctrlBase.Router, http.MethodPost, "/api/v1/namespaces", `{not-json`)

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("returns 500 when the usecase fails", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("CreateNamespace", mock.Anything, mock.Anything).Return(nil, errBoom)

		recorder := doReq(t, ctrlBase.Router, http.MethodPost, "/api/v1/namespaces", `{"metadata":{"name":"prod"}}`)

		require.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}

func TestNamespaceController_Update(t *testing.T) {
	t.Parallel()

	t.Run("updates a namespace", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("UpdateNamespace", mock.Anything, "prod", mock.Anything).Return(newNamespace("prod"), nil)

		recorder := doReq(t, ctrlBase.Router, http.MethodPut, "/api/v1/namespaces/prod", `{"metadata":{"name":"prod"}}`)

		require.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("returns 400 on an invalid body", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := setup(t)

		recorder := doReq(t, ctrlBase.Router, http.MethodPut, "/api/v1/namespaces/prod", `{not-json`)

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("returns 404 when the namespace does not exist", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("UpdateNamespace", mock.Anything, "missing", mock.Anything).Return(nil, model.ErrResourceNotExist)

		recorder := doReq(t, ctrlBase.Router, http.MethodPut, "/api/v1/namespaces/missing", `{"metadata":{"name":"missing"}}`)

		require.Equal(t, http.StatusNotFound, recorder.Code)
	})
}

func TestNamespaceController_Delete(t *testing.T) {
	t.Parallel()

	t.Run("deletes a namespace", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("DeleteNamespace", mock.Anything, "prod").Return(nil)

		recorder := doReq(t, ctrlBase.Router, http.MethodDelete, "/api/v1/namespaces/prod", "")

		require.Equal(t, http.StatusNoContent, recorder.Code)
	})

	t.Run("returns 404 when the namespace does not exist", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("DeleteNamespace", mock.Anything, "missing").Return(model.ErrResourceNotExist)

		recorder := doReq(t, ctrlBase.Router, http.MethodDelete, "/api/v1/namespaces/missing", "")

		require.Equal(t, http.StatusNotFound, recorder.Code)
	})
}

// TestNamespaceController_MissingNamespaceParam covers the required-path-param validation
// branches in Get/Update/Delete, which cannot be reached through routing (the :namespace
// segment is never empty when matched).
func TestNamespaceController_MissingNamespaceParam(t *testing.T) {
	t.Parallel()

	handlers := map[string]func(*namespace.Controller) gin.HandlerFunc{
		"Get":    func(c *namespace.Controller) gin.HandlerFunc { return c.Get },
		"Update": func(c *namespace.Controller) gin.HandlerFunc { return c.Update },
		"Delete": func(c *namespace.Controller) gin.HandlerFunc { return c.Delete },
	}

	for name, pick := range handlers {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			controller := namespace.NewController(newMockNamespaceUsecase(t), slog.Default())

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
