package server_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"go.uber.org/goleak"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/server"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

// errBoom is a generic error used to drive the usecase into its failure path.
var errBoom = errors.New("boom")

// mockServerUsecase is a testify mock of usecase.ServerManageUsecase.
type mockServerUsecase struct {
	mock.Mock
}

func newMockServerUsecase(t *testing.T) *mockServerUsecase {
	t.Helper()

	m := &mockServerUsecase{}
	m.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}

func (m *mockServerUsecase) ListServers(ctx context.Context) (*v1.ListResponse[v1.Server], error) {
	args := m.Called(ctx)

	res, _ := args.Get(0).(*v1.ListResponse[v1.Server])

	return res, args.Error(1) //nolint:wrapcheck // mock error
}

func TestServerController_RoutesInfo(t *testing.T) {
	t.Parallel()

	controller := server.NewController(slog.Default(), newMockServerUsecase(t))

	routes := controller.RoutesInfo()
	require.Len(t, routes, 1)
	assert.Equal(t, http.MethodGet, routes[0].Method)
	assert.Equal(t, "/api/v1/servers", routes[0].Path)
	assert.NotNil(t, routes[0].HandlerFunc)
}

func TestServerController_List(t *testing.T) {
	t.Parallel()

	t.Run("returns the list of servers", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		usecase := newMockServerUsecase(t)
		controller := server.NewController(slog.Default(), usecase)
		ctrlBase.SetupRouter(controller)

		servers := []v1.Server{{ID: "server-1"}, {ID: "server-2"}}
		usecase.On("ListServers", mock.Anything).
			Return(v1.NewServerListResponse(servers, v1.ListMeta{RemainingItemCount: 0, Continue: ""}), nil)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/servers", nil)
		require.NoError(t, err)
		ctrlBase.Router.ServeHTTP(recorder, req)

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, int64(2), gjson.Get(recorder.Body.String(), "items.#").Int())
		assert.Equal(t, "server-1", gjson.Get(recorder.Body.String(), "items.0.id").String())
	})

	t.Run("returns 500 when the usecase fails", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		usecase := newMockServerUsecase(t)
		controller := server.NewController(slog.Default(), usecase)
		ctrlBase.SetupRouter(controller)

		usecase.On("ListServers", mock.Anything).Return(nil, errBoom)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/servers", nil)
		require.NoError(t, err)
		ctrlBase.Router.ServeHTTP(recorder, req)

		require.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}
