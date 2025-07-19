package command_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"go.uber.org/goleak"

	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/command"
	"github.com/minuk-dev/opampcommander/internal/application/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestCommandController_Get(t *testing.T) {
	t.Parallel()

	t.Run("Get Command - happycase", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		commandUsecase := newMockCommandUsecase(t)
		controller := command.NewController(commandUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// given
		commandID := uuid.New()
		//exhaustruct:ignore
		commandData := &model.Command{
			ID: commandID,
		}
		commandUsecase.On("GetCommand", mock.Anything, commandID).Return(commandData, nil)

		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/commands/"+commandID.String(), nil)
		require.NoError(t, err)

		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "application/json; charset=utf-8", recorder.Header().Get("Content-Type"))
		assert.Equal(t, commandID.String(), gjson.Get(recorder.Body.String(), "id").String())
	})

	t.Run("Get Command - 400 Bad Request when invalid ID", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		commandUsecase := newMockCommandUsecase(t)
		controller := command.NewController(commandUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/commands/invalid-id", nil)
		require.NoError(t, err)

		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("Get Command - 500 Internal Server Error when usecase fails", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		commandUsecase := newMockCommandUsecase(t)
		controller := command.NewController(commandUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// given
		commandID := uuid.New()
		commandUsecase.On("GetCommand", mock.Anything, commandID).Return((*model.Command)(nil), assert.AnError)

		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/commands/"+commandID.String(), nil)
		require.NoError(t, err)

		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}

func TestCommandController_List(t *testing.T) {
	t.Parallel()

	t.Run("List Commands - happycase", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		commandUsecase := newMockCommandUsecase(t)
		controller := command.NewController(commandUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// given
		commands := []*model.Command{
			{ID: uuid.New()},
			{ID: uuid.New()},
		}
		commandUsecase.On("ListCommands", mock.Anything, mock.Anything).Return(&model.ListResponse[*model.Command]{
			RemainingItemCount: 0,
			Continue:           "",
			Items:              commands,
		}, nil)

		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/commands", nil)
		require.NoError(t, err)

		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "application/json; charset=utf-8", recorder.Header().Get("Content-Type"))
		assert.Equal(t, len(commands), int(gjson.Get(recorder.Body.String(), "items.#").Int()))
	})

	t.Run("List Commands - 400 Bad Request when invalid continue query parameters", func(t *testing.T) {
		t.Parallel()
		ctrlBase := testutil.NewBase(t).ForController()
		commandUsecase := newMockCommandUsecase(t)
		controller := command.NewController(commandUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/commands?limit=invalid", nil)
		require.NoError(t, err)

		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
		assert.Equal(t, "application/json; charset=utf-8", recorder.Header().Get("Content-Type"))
		assert.Equal(t, "invalid limit parameter", gjson.Get(recorder.Body.String(), "error").String())
	})

	t.Run("List Commands - 400 Bad Request when invalid limit query parameters", func(t *testing.T) {
		t.Parallel()
		ctrlBase := testutil.NewBase(t).ForController()
		commandUsecase := newMockCommandUsecase(t)
		controller := command.NewController(commandUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router
		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/commands?limit=invalid", nil)
		require.NoError(t, err)

		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
		assert.Equal(t, "application/json; charset=utf-8", recorder.Header().Get("Content-Type"))
		assert.Equal(t, "invalid limit parameter", gjson.Get(recorder.Body.String(), "error").String())
	})

	t.Run("List Commands - 500 Internal Server Error when usecase fails", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		commandUsecase := newMockCommandUsecase(t)
		controller := command.NewController(commandUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// given
		commandUsecase.On("ListCommands", mock.Anything, mock.Anything).
			Return((*model.ListResponse[*model.Command])(nil), assert.AnError)

		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/commands", nil)
		require.NoError(t, err)

		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}

var _ port.CommandLookUpUsecase = (*mockCommandUsecase)(nil)

type mockCommandUsecase struct {
	mock.Mock
}

func newMockCommandUsecase(t *testing.T) *mockCommandUsecase {
	t.Helper()

	//exhaustruct:ignore
	return &mockCommandUsecase{}
}

//nolint:forcetypeassert,wrapcheck
func (m *mockCommandUsecase) GetCommand(ctx context.Context, commandID uuid.UUID) (*model.Command, error) {
	args := m.Called(ctx, commandID)

	return args.Get(0).(*model.Command), args.Error(1)
}

//nolint:wrapcheck,forcetypeassert
func (m *mockCommandUsecase) ListCommands(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*model.Command], error) {
	args := m.Called(ctx, options)

	return args.Get(0).(*model.ListResponse[*model.Command]), args.Error(1)
}

//nolint:forcetypeassert,wrapcheck
func (m *mockCommandUsecase) GetCommandByInstanceUID(
	ctx context.Context,
	instanceUID uuid.UUID,
) ([]*model.Command, error) {
	args := m.Called(ctx, instanceUID)

	return args.Get(0).([]*model.Command), args.Error(1)
}
