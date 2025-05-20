package agent_test

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

	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/agent"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

//nolint:funlen
func TestAgentControllerListAgent(t *testing.T) {
	t.Parallel()

	t.Run("List Agents - happycase", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := newMockAgentUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		// given
		instanceUIDs := []uuid.UUID{uuid.New(), uuid.New()}
		agents := []*model.Agent{
			{
				InstanceUID: instanceUIDs[0],
			},
			{
				InstanceUID: instanceUIDs[1],
			},
		}
		agentUsecase.On("ListAgents", mock.Anything).Return(agents, nil)

		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/api/v1/agents", nil)
		require.NoError(t, err)

		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "application/json; charset=utf-8", recorder.Header().Get("Content-Type"))
		assert.Equal(t, int64(2), gjson.Get(recorder.Body.String(), "#").Int())
		assert.Equal(t, instanceUIDs[0].String(), gjson.Get(recorder.Body.String(), "0.instanceUid").String())
		assert.Equal(t, instanceUIDs[1].String(), gjson.Get(recorder.Body.String(), "1.instanceUid").String())
	})

	t.Run("List Agents - empty returns 200, empty", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := newMockAgentUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		// given
		agentUsecase.On("ListAgents", mock.Anything).Return([]*model.Agent{}, nil)

		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/api/v1/agents", nil)
		require.NoError(t, err)

		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.JSONEq(t, "[]", recorder.Body.String())
	})

	t.Run("List Agents - any error returns 500", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := newMockAgentUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// given
		agentUsecase.On("ListAgents", mock.Anything).Return(nil, assert.AnError)
		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/api/v1/agents", nil)
		require.NoError(t, err)
		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}

//nolint:funlen
func TestAgentControllerGetAgent(t *testing.T) {
	t.Parallel()
	t.Run("Get Agent - happycase", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := newMockAgentUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// given
		instanceUID := uuid.New()
		//exhaustruct:ignore
		agentData := &model.Agent{
			InstanceUID: instanceUID,
		}
		agentUsecase.On("GetAgent", mock.Anything, instanceUID).Return(agentData, nil)
		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/api/v1/agents/"+instanceUID.String(), nil)
		require.NoError(t, err)
		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "application/json; charset=utf-8", recorder.Header().Get("Content-Type"))
		t.Logf("response body: %s", recorder.Body.String())
		assert.Equal(t, instanceUID.String(), gjson.Get(recorder.Body.String(), "instanceUid").String())
	})

	t.Run("Get Agent - not found error returns 404", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := newMockAgentUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// given
		instanceUID := uuid.New()

		agentUsecase.On("GetAgent", mock.Anything, mock.Anything).Return((*model.Agent)(nil), port.ErrAgentNotExist)
		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/api/v1/agents/"+instanceUID.String(), nil)
		require.NoError(t, err)
		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusNotFound, recorder.Code)
		assert.JSONEq(t, `{"error":"agent not found"}`, recorder.Body.String())
	})

	t.Run("Get Agent - instanceUID is not uuid returns 400", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := newMockAgentUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/agents/not-a-uuid", nil)
		require.NoError(t, err)
		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("Get Agent - other error returns 500", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := newMockAgentUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// given
		instanceUID := uuid.New()
		agentUsecase.On("GetAgent", mock.Anything, instanceUID).Return(nil, assert.AnError)
		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/api/v1/agents/"+instanceUID.String(), nil)
		require.NoError(t, err)
		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}

var _ port.AgentUsecase = (*mockAgentUsecase)(nil)

func newMockAgentUsecase(t *testing.T) *mockAgentUsecase {
	t.Helper()

	//exhaustruct:ignore
	return &mockAgentUsecase{}
}

type mockAgentUsecase struct {
	mock.Mock
}

//nolint:wrapcheck,forcetypeassert
func (m *mockAgentUsecase) GetAgent(ctx context.Context, instanceUID uuid.UUID) (*model.Agent, error) {
	args := m.Called(ctx, instanceUID)

	return args.Get(0).(*model.Agent), args.Error(1)
}

//nolint:wrapcheck,forcetypeassert
func (m *mockAgentUsecase) GetOrCreateAgent(ctx context.Context, instanceUID uuid.UUID) (*model.Agent, error) {
	args := m.Called(ctx, instanceUID)

	return args.Get(0).(*model.Agent), args.Error(1)
}

//nolint:wrapcheck
func (m *mockAgentUsecase) SaveAgent(ctx context.Context, agent *model.Agent) error {
	args := m.Called(ctx, agent)

	return args.Error(0)
}

//nolint:wrapcheck,forcetypeassert
func (m *mockAgentUsecase) ListAgents(ctx context.Context) ([]*model.Agent, error) {
	args := m.Called(ctx)

	return args.Get(0).([]*model.Agent), args.Error(1)
}

//nolint:wrapcheck
func (m *mockAgentUsecase) UpdateAgentConfig(ctx context.Context, instanceUID uuid.UUID, config any) error {
	args := m.Called(ctx, instanceUID, config)

	return args.Error(0)
}
