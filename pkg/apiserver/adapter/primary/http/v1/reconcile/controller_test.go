package reconcile_test

import (
	"context"
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

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/reconcile"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	goleak.VerifyTestMain(m)
}

// mockReconcileUsecase is a testify mock of usecase.ReconcileManageUsecase.
type mockReconcileUsecase struct {
	mock.Mock
}

func newMockReconcileUsecase(t *testing.T) *mockReconcileUsecase {
	t.Helper()

	m := &mockReconcileUsecase{}
	m.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}

func (m *mockReconcileUsecase) Reconcile(ctx context.Context, kind, namespace, name string) error {
	args := m.Called(ctx, kind, namespace, name)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func (m *mockReconcileUsecase) ReconcileKinds(ctx context.Context) []string {
	args := m.Called(ctx)

	res, _ := args.Get(0).([]string)

	return res
}

func TestReconcileController_RoutesInfo(t *testing.T) {
	t.Parallel()

	controller := reconcile.NewController(newMockReconcileUsecase(t), slog.Default())

	routes := controller.RoutesInfo()
	require.Len(t, routes, 2)

	got := make(map[string]struct{}, len(routes))
	for _, route := range routes {
		got[route.Method+" "+route.Path] = struct{}{}

		assert.NotNil(t, route.HandlerFunc)
	}

	assert.Contains(t, got, "POST /api/v1/namespaces/:namespace/reconcile/:kind/:name")
	assert.Contains(t, got, "GET /api/v1/reconcile/kinds")
}

func TestReconcileController_Reconcile(t *testing.T) {
	t.Parallel()

	t.Run("returns 204 on success", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		usecase := newMockReconcileUsecase(t)
		controller := reconcile.NewController(usecase, slog.Default())
		ctrlBase.SetupRouter(controller)

		usecase.On("Reconcile", mock.Anything, "agent", "default", "uid-1").Return(nil)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
			"/api/v1/namespaces/default/reconcile/agent/uid-1", nil)
		require.NoError(t, err)
		ctrlBase.Router.ServeHTTP(recorder, req)

		require.Equal(t, http.StatusNoContent, recorder.Code)
	})

	t.Run("returns 404 when the resource does not exist", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		usecase := newMockReconcileUsecase(t)
		controller := reconcile.NewController(usecase, slog.Default())
		ctrlBase.SetupRouter(controller)

		usecase.On("Reconcile", mock.Anything, "agentgroup", "default", "missing").
			Return(model.ErrResourceNotExist)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
			"/api/v1/namespaces/default/reconcile/agentgroup/missing", nil)
		require.NoError(t, err)
		ctrlBase.Router.ServeHTTP(recorder, req)

		require.Equal(t, http.StatusNotFound, recorder.Code)
	})

	t.Run("returns 400 on an invalid argument", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		usecase := newMockReconcileUsecase(t)
		controller := reconcile.NewController(usecase, slog.Default())
		ctrlBase.SetupRouter(controller)

		usecase.On("Reconcile", mock.Anything, "unknownkind", "default", "x").
			Return(model.ErrInvalidArgument)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
			"/api/v1/namespaces/default/reconcile/unknownkind/x", nil)
		require.NoError(t, err)
		ctrlBase.Router.ServeHTTP(recorder, req)

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	// The :namespace/:kind/:name path segments can never be empty when routed, so the required-param
	// validation branches are exercised by invoking the handler directly with a missing param.
	t.Run("returns 400 when a required path param is missing", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name   string
			params gin.Params
		}{
			{"missing namespace", gin.Params{{Key: "kind", Value: "agent"}, {Key: "name", Value: "n"}}},
			{"missing kind", gin.Params{{Key: "namespace", Value: "default"}, {Key: "name", Value: "n"}}},
			{"missing name", gin.Params{{Key: "namespace", Value: "default"}, {Key: "kind", Value: "agent"}}},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				controller := reconcile.NewController(newMockReconcileUsecase(t), slog.Default())

				recorder := httptest.NewRecorder()
				ginCtx, _ := gin.CreateTestContext(recorder)
				req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "/", nil)
				require.NoError(t, err)

				ginCtx.Request = req
				ginCtx.Params = tc.params

				controller.Reconcile(ginCtx)

				require.Equal(t, http.StatusBadRequest, recorder.Code)
			})
		}
	})
}

func TestReconcileController_ListKinds(t *testing.T) {
	t.Parallel()

	ctrlBase := testutil.NewBase(t).ForController()
	usecase := newMockReconcileUsecase(t)
	controller := reconcile.NewController(usecase, slog.Default())
	ctrlBase.SetupRouter(controller)

	usecase.On("ReconcileKinds", mock.Anything).Return([]string{"agent", "agentgroup", "agentremoteconfig"})

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/reconcile/kinds", nil)
	require.NoError(t, err)
	ctrlBase.Router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, int64(3), gjson.Get(recorder.Body.String(), "#").Int())
	assert.Equal(t, "agent", gjson.Get(recorder.Body.String(), "0").String())
}
