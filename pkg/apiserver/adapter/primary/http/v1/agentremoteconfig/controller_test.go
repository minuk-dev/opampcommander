package agentremoteconfig_test

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
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/agentremoteconfig"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	goleak.VerifyTestMain(m)
}

var errBoom = errors.New("boom")

// mockUsecase is a testify mock of usecase.AgentRemoteConfigManageUsecase.
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

func (m *mockUsecase) GetAgentRemoteConfig(
	ctx context.Context, namespace, name string, options *port.GetOptions,
) (*v1.AgentRemoteConfig, error) {
	args := m.Called(ctx, namespace, name, options)

	res, _ := args.Get(0).(*v1.AgentRemoteConfig)

	return res, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockUsecase) ListAgentRemoteConfigs(
	ctx context.Context, options *port.ListOptions,
) (*v1.ListResponse[v1.AgentRemoteConfig], error) {
	args := m.Called(ctx, options)

	res, _ := args.Get(0).(*v1.ListResponse[v1.AgentRemoteConfig])

	return res, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockUsecase) CreateAgentRemoteConfig(
	ctx context.Context, arc *v1.AgentRemoteConfig,
) (*v1.AgentRemoteConfig, error) {
	args := m.Called(ctx, arc)

	res, _ := args.Get(0).(*v1.AgentRemoteConfig)

	return res, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockUsecase) UpdateAgentRemoteConfig(
	ctx context.Context, namespace, name string, arc *v1.AgentRemoteConfig,
) (*v1.AgentRemoteConfig, error) {
	args := m.Called(ctx, namespace, name, arc)

	res, _ := args.Get(0).(*v1.AgentRemoteConfig)

	return res, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockUsecase) DeleteAgentRemoteConfig(ctx context.Context, namespace, name string) error {
	args := m.Called(ctx, namespace, name)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func newConfig() *v1.AgentRemoteConfig {
	//exhaustruct:ignore
	return &v1.AgentRemoteConfig{
		Kind:       v1.AgentRemoteConfigKind,
		APIVersion: v1.APIVersion,
		//exhaustruct:ignore
		Metadata: v1.AgentRemoteConfigMetadata{Name: "cfg", Namespace: "default"},
	}
}

func setup(t *testing.T) (*testutil.ControllerBase, *mockUsecase) {
	t.Helper()

	ctrlBase := testutil.NewBase(t).ForController()
	usecase := newMockUsecase(t)
	controller := agentremoteconfig.NewController(usecase, slog.Default())
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

const base = "/api/v1/namespaces/default/agentremoteconfigs"

func TestController_RoutesInfo(t *testing.T) {
	t.Parallel()

	controller := agentremoteconfig.NewController(newMockUsecase(t), slog.Default())

	routes := controller.RoutesInfo()
	require.Len(t, routes, 5)

	got := make(map[string]struct{}, len(routes))
	for _, route := range routes {
		got[route.Method+" "+route.Path] = struct{}{}

		assert.NotNil(t, route.HandlerFunc)
	}

	for _, want := range []string{
		"GET /api/v1/namespaces/:namespace/agentremoteconfigs",
		"GET /api/v1/namespaces/:namespace/agentremoteconfigs/:name",
		"POST /api/v1/namespaces/:namespace/agentremoteconfigs",
		"PUT /api/v1/namespaces/:namespace/agentremoteconfigs/:name",
		"DELETE /api/v1/namespaces/:namespace/agentremoteconfigs/:name",
	} {
		assert.Contains(t, got, want)
	}
}

func TestController_List(t *testing.T) {
	t.Parallel()

	t.Run("returns the list", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		//exhaustruct:ignore
		usecase.On("ListAgentRemoteConfigs", mock.Anything, mock.Anything).
			Return(&v1.ListResponse[v1.AgentRemoteConfig]{Items: []v1.AgentRemoteConfig{*newConfig()}}, nil)

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
		usecase.On("ListAgentRemoteConfigs", mock.Anything, mock.Anything).Return(nil, errBoom)

		recorder := doReq(t, ctrlBase.Router, http.MethodGet, base, "")

		require.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}

func TestController_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns the config", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("GetAgentRemoteConfig", mock.Anything, "default", "cfg", mock.Anything).Return(newConfig(), nil)

		recorder := doReq(t, ctrlBase.Router, http.MethodGet, base+"/cfg", "")

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "cfg", gjson.Get(recorder.Body.String(), "metadata.name").String())
	})

	t.Run("returns 400 on an invalid includeDeleted", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := setup(t)

		recorder := doReq(t, ctrlBase.Router, http.MethodGet, base+"/cfg?includeDeleted=nope", "")

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("returns 404 when the config does not exist", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("GetAgentRemoteConfig", mock.Anything, "default", "missing", mock.Anything).
			Return(nil, model.ErrResourceNotExist)

		recorder := doReq(t, ctrlBase.Router, http.MethodGet, base+"/missing", "")

		require.Equal(t, http.StatusNotFound, recorder.Code)
	})
}

func TestController_Create(t *testing.T) {
	t.Parallel()

	t.Run("creates a config and sets the Location header", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("CreateAgentRemoteConfig", mock.Anything, mock.Anything).Return(newConfig(), nil)

		recorder := doReq(t, ctrlBase.Router, http.MethodPost, base, `{"metadata":{"name":"cfg"}}`)

		require.Equal(t, http.StatusCreated, recorder.Code)
		assert.Equal(t, base+"/cfg", recorder.Header().Get("Location"))
	})

	t.Run("returns 400 on an invalid body", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := setup(t)

		recorder := doReq(t, ctrlBase.Router, http.MethodPost, base, `{not-json`)

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("returns 500 when the usecase fails", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("CreateAgentRemoteConfig", mock.Anything, mock.Anything).Return(nil, errBoom)

		recorder := doReq(t, ctrlBase.Router, http.MethodPost, base, `{"metadata":{"name":"cfg"}}`)

		require.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}

func TestController_Update(t *testing.T) {
	t.Parallel()

	t.Run("updates a config", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("UpdateAgentRemoteConfig", mock.Anything, "default", "cfg", mock.Anything).Return(newConfig(), nil)

		recorder := doReq(t, ctrlBase.Router, http.MethodPut, base+"/cfg", `{"metadata":{"name":"cfg"}}`)

		require.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("returns 400 on an invalid body", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := setup(t)

		recorder := doReq(t, ctrlBase.Router, http.MethodPut, base+"/cfg", `{not-json`)

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("returns 404 when the config does not exist", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("UpdateAgentRemoteConfig", mock.Anything, "default", "missing", mock.Anything).
			Return(nil, model.ErrResourceNotExist)

		recorder := doReq(t, ctrlBase.Router, http.MethodPut, base+"/missing", `{"metadata":{"name":"missing"}}`)

		require.Equal(t, http.StatusNotFound, recorder.Code)
	})
}

func TestController_Delete(t *testing.T) {
	t.Parallel()

	t.Run("deletes a config", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("DeleteAgentRemoteConfig", mock.Anything, "default", "cfg").Return(nil)

		recorder := doReq(t, ctrlBase.Router, http.MethodDelete, base+"/cfg", "")

		require.Equal(t, http.StatusNoContent, recorder.Code)
	})

	t.Run("returns 404 when the config does not exist", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("DeleteAgentRemoteConfig", mock.Anything, "default", "missing").Return(model.ErrResourceNotExist)

		recorder := doReq(t, ctrlBase.Router, http.MethodDelete, base+"/missing", "")

		require.Equal(t, http.StatusNotFound, recorder.Code)
	})
}

// TestController_MissingParams covers the required :namespace / :name validation branches in
// Get/Create/Update/Delete, unreachable through routing since the segments are never empty.
func TestController_MissingParams(t *testing.T) {
	t.Parallel()

	controller := agentremoteconfig.NewController(newMockUsecase(t), slog.Default())
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
