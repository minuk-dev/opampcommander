package connection_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"go.uber.org/goleak"

	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/connection"
	applicationport "github.com/minuk-dev/opampcommander/internal/application/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
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
		controller := connection.NewController(adminUsecase)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// given
		connections := []*model.Connection{
			{
				UID:                uuid.New(),
				InstanceUID:        uuid.New(),
				LastCommunicatedAt: time.Now(),
			},
			{
				UID:                uuid.New(),
				InstanceUID:        uuid.New(),
				LastCommunicatedAt: time.Now(),
			},
		}
		adminUsecase.On("ListConnections", mock.Anything, mock.Anything).
			Return(&model.ListResponse[*model.Connection]{
				RemainingItemCount: 0,
				Continue:           "",
				Items:              connections,
			}, nil)

		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/connections", nil)
		require.NoError(t, err)

		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "application/json; charset=utf-8", recorder.Header().Get("Content-Type"))
		t.Logf("Response Body: %s", recorder.Body.String())
		assert.Equal(t, len(connections), int(gjson.Get(recorder.Body.String(), "items.#").Int()))
	})

	t.Run("List Connections - error case", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		adminUsecase := newMockAdminUsecase(t)
		controller := connection.NewController(adminUsecase)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// given
		adminUsecase.On("ListConnections", mock.Anything, mock.Anything).
			Return((*model.ListResponse[*model.Connection])(nil), assert.AnError)

		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/connections", nil)
		require.NoError(t, err)

		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	})

	t.Run("List Connections - invalid limit query parameter", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		adminUsecase := newMockAdminUsecase(t)
		controller := connection.NewController(adminUsecase)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/connections?limit=invalid", nil)
		require.NoError(t, err)

		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})
}

var _ applicationport.AdminUsecase = (*mockAdminUsecase)(nil)

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
	options *model.ListOptions,
) (*model.ListResponse[*model.Connection], error) {
	args := m.Called(ctx, options)

	return args.Get(0).(*model.ListResponse[*model.Connection]), args.Error(1)
}

//nolint:wrapcheck
func (m *mockAdminUsecase) ApplyRawConfig(ctx context.Context, targetInstanceUID uuid.UUID, config any) error {
	args := m.Called(ctx, targetInstanceUID, config)

	return args.Error(0)
}
