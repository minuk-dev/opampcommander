package connection_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"go.uber.org/goleak"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/connection"
	applicationport "github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/usecase"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestConnectionController_List(t *testing.T) {
	t.Parallel()

	t.Run("List Connections - happycase", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		adminUsecase := newMockAdminUsecase(t)
		controller := connection.NewController(slog.Default(), adminUsecase)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// given
		conn1UID := uuid.New()
		conn2UID := uuid.New()
		agent1UID := uuid.New()
		agent2UID := uuid.New()

		connections := []v1.Connection{
			{
				ID:          conn1UID,
				InstanceUID: agent1UID,
				Namespace:   "default",
				Type:        "WebSocket",
				Alive:       true,
			},
			{
				ID:          conn2UID,
				InstanceUID: agent2UID,
				Namespace:   "default",
				Type:        "HTTP",
				Alive:       true,
			},
		}
		adminUsecase.On("ListConnections", mock.Anything, "default", mock.Anything).
			Return(v1.NewConnectionListResponse(connections, v1.ListMeta{
				RemainingItemCount: 0,
				Continue:           "",
			}), nil)

		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(
			t.Context(), http.MethodGet, "/api/v1/namespaces/default/connections", nil,
		)
		require.NoError(t, err)

		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "application/json; charset=utf-8", recorder.Header().Get("Content-Type"))
		t.Logf("Response Body: %s", recorder.Body.String())

		// Verify connection count
		assert.Equal(t, len(connections), int(gjson.Get(recorder.Body.String(), "items.#").Int()))

		// Verify connection types are correctly returned
		items := gjson.Get(recorder.Body.String(), "items").Array()
		assert.Len(t, items, 2)
		assert.Equal(t, "WebSocket", gjson.Get(items[0].String(), "type").String())
		assert.Equal(t, "HTTP", gjson.Get(items[1].String(), "type").String())
	})

	t.Run("List Connections - error case", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		adminUsecase := newMockAdminUsecase(t)
		controller := connection.NewController(slog.Default(), adminUsecase)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// given
		adminUsecase.On("ListConnections", mock.Anything, "default", mock.Anything).
			Return((*v1.ListResponse[v1.Connection])(nil), assert.AnError)

		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(
			t.Context(), http.MethodGet, "/api/v1/namespaces/default/connections", nil,
		)
		require.NoError(t, err)

		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	})

	t.Run("List Connections - invalid limit query parameter", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		adminUsecase := newMockAdminUsecase(t)
		controller := connection.NewController(slog.Default(), adminUsecase)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			"/api/v1/namespaces/default/connections?limit=invalid",
			nil,
		)
		require.NoError(t, err)

		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})
}

var _ usecase.AdminUsecase = (*mockAdminUsecase)(nil)

type mockAdminUsecase struct {
	mock.Mock
}

func newMockAdminUsecase(t *testing.T) *mockAdminUsecase {
	t.Helper()

	//exhaustruct:ignore
	return &mockAdminUsecase{}
}

//nolint:forcetypeassert,wrapcheck
func (m *mockAdminUsecase) ListConnections(
	ctx context.Context,
	namespace string,
	options *applicationport.ListOptions,
) (*v1.ListResponse[v1.Connection], error) {
	args := m.Called(ctx, namespace, options)

	return args.Get(0).(*v1.ListResponse[v1.Connection]), args.Error(1)
}

func (m *mockAdminUsecase) ListClusterConnections(
	ctx context.Context,
	namespace string,
	serverID string,
	options *applicationport.ListOptions,
) (*v1.ListResponse[v1.Connection], error) {
	args := m.Called(ctx, namespace, serverID, options)
	resp, _ := args.Get(0).(*v1.ListResponse[v1.Connection])

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}
