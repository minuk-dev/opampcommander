package agent_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"go.uber.org/goleak"

	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/agent"
	applicationport "github.com/minuk-dev/opampcommander/internal/application/port"
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
		agentUsecase := newMockAgentManageUsecase(t)
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
		agentUsecase := newMockAgentManageUsecase(t)
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
		agentUsecase := newMockAgentManageUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// given
		agentUsecase.On("ListAgents", mock.Anything).Return(([]*model.Agent)(nil), assert.AnError)
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
		agentUsecase := newMockAgentManageUsecase(t)
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
		agentUsecase := newMockAgentManageUsecase(t)
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
		agentUsecase := newMockAgentManageUsecase(t)
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
		agentUsecase := newMockAgentManageUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// given
		instanceUID := uuid.New()
		agentUsecase.On("GetAgent", mock.Anything, instanceUID).Return((*model.Agent)(nil), assert.AnError)
		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/api/v1/agents/"+instanceUID.String(), nil)
		require.NoError(t, err)
		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}

//nolint:funlen
func TestAgentController_UpdateAgentConfig(t *testing.T) {
	t.Parallel()

	t.Run("Update Agent Config - happycase", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentManageUsecase := newMockAgentManageUsecase(t)
		controller := agent.NewController(agentManageUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// given
		requestBody := `{"targetInstanceUid":"` + uuid.New().String() + `","remoteConfig":{"key":"value"}}`

		agentManageUsecase.On("SendCommand", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(ctx, http.MethodPost,
			fmt.Sprintf("/api/v1/agents/%s/update-agent-config", uuid.New().String()),
			strings.NewReader(requestBody))
		require.NoError(t, err)

		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusCreated, recorder.Code)
	})

	t.Run("Update Agent Config - 400 Bad Request when invalid request body", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentManageUsecase := newMockAgentManageUsecase(t)
		controller := agent.NewController(agentManageUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(ctx, http.MethodPost,
			fmt.Sprintf("/api/v1/agents/%s/update-agent-config", uuid.New().String()),
			strings.NewReader("invalid request body"))
		require.NoError(t, err)

		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("Update Agent Config - 500 Internal Server Error when usecase fails", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentManageUsecase := newMockAgentManageUsecase(t)
		controller := agent.NewController(agentManageUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// given
		requestBody := `{"targetInstanceUid":"` + uuid.New().String() + `","remoteConfig":{"key":"value"}}`

		agentManageUsecase.On("SendCommand", mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError)

		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(ctx, http.MethodPost,
			fmt.Sprintf("/api/v1/agents/%s/update-agent-config", uuid.New().String()),
			strings.NewReader(requestBody))
		require.NoError(t, err)

		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}

var _ applicationport.AgentManageUsecase = (*mockAgentManageUsecase)(nil)

func newMockAgentManageUsecase(t *testing.T) *mockAgentManageUsecase {
	t.Helper()

	//exhaustruct:ignore
	return &mockAgentManageUsecase{}
}

type mockAgentManageUsecase struct {
	mock.Mock
}

//nolint:wrapcheck,forcetypeassert
func (m *mockAgentManageUsecase) GetAgent(ctx context.Context, instanceUID uuid.UUID) (*model.Agent, error) {
	args := m.Called(ctx, instanceUID)

	return args.Get(0).(*model.Agent), args.Error(1)
}

//nolint:wrapcheck,forcetypeassert
func (m *mockAgentManageUsecase) GetOrCreateAgent(ctx context.Context, instanceUID uuid.UUID) (*model.Agent, error) {
	args := m.Called(ctx, instanceUID)

	return args.Get(0).(*model.Agent), args.Error(1)
}

//nolint:wrapcheck
func (m *mockAgentManageUsecase) SaveAgent(ctx context.Context, agent *model.Agent) error {
	args := m.Called(ctx, agent)

	return args.Error(0)
}

//nolint:wrapcheck,forcetypeassert
func (m *mockAgentManageUsecase) ListAgents(ctx context.Context) ([]*model.Agent, error) {
	args := m.Called(ctx)

	return args.Get(0).([]*model.Agent), args.Error(1)
}

//nolint:wrapcheck
func (m *mockAgentManageUsecase) SendCommand(ctx context.Context, instanceUID uuid.UUID, command *model.Command) error {
	args := m.Called(ctx, instanceUID, command)

	return args.Error(0)
}
