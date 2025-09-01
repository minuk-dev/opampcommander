package agent_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"go.uber.org/goleak"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	v1agent "github.com/minuk-dev/opampcommander/api/v1/agent"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/agent"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/agent/usecasemock"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestAgentControllerListAgent(t *testing.T) {
	t.Parallel()

	t.Run("List Agents - happycase", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := usecasemock.NewMockAgentManageUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router
		// given
		instanceUIDs := []uuid.UUID{uuid.New(), uuid.New()}
		//exhaustruct:ignore
		agents := []v1agent.Agent{
			{
				InstanceUID: instanceUIDs[0],
			},
			{
				InstanceUID: instanceUIDs[1],
			},
		}
		agentUsecase.EXPECT().
			ListAgents(mock.Anything, mock.Anything).
			Return(&v1agent.ListResponse{
				APIVersion: "v1",
				Kind:       v1agent.AgentKind,
				Items:      agents,
				Metadata: v1.ListMeta{
					RemainingItemCount: 0,
					Continue:           "",
				},
			}, nil)

		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/agents", nil)
		require.NoError(t, err)

		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "application/json; charset=utf-8", recorder.Header().Get("Content-Type"))
		assert.Equal(t, int64(2), gjson.Get(recorder.Body.String(), "items.#").Int())
		assert.Equal(t, instanceUIDs[0].String(), gjson.Get(recorder.Body.String(), "items.0.instanceUid").String())
		assert.Equal(t, instanceUIDs[1].String(), gjson.Get(recorder.Body.String(), "items.1.instanceUid").String())
	})

	t.Run("List Agents - empty returns 200, empty", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := usecasemock.NewMockAgentManageUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// given
		agentUsecase.EXPECT().
			ListAgents(mock.Anything, mock.Anything).
			Return(&v1agent.ListResponse{
				APIVersion: "v1",
				Kind:       v1agent.AgentKind,
				Items:      []v1agent.Agent{},
				Metadata: v1.ListMeta{
					RemainingItemCount: 0,
					Continue:           "",
				},
			}, nil)

		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/agents", nil)
		require.NoError(t, err)

		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "application/json; charset=utf-8", recorder.Header().Get("Content-Type"))
		assert.Equal(t, int64(0), gjson.Get(recorder.Body.String(), "items.#").Int())
	})

	t.Run("List Agents - invalid limit returns 400", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := usecasemock.NewMockAgentManageUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router
		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/agents?limit=invalid", nil)
		require.NoError(t, err)
		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
		assert.JSONEq(t, `{"error":"invalid limit parameter"}`, recorder.Body.String())
	})

	t.Run("List Agents - any error returns 500", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := usecasemock.NewMockAgentManageUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// given
		agentUsecase.EXPECT().
			ListAgents(mock.Anything, mock.Anything).
			Return(nil, assert.AnError)
		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/agents", nil)
		require.NoError(t, err)
		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}

func TestAgentControllerGetAgent(t *testing.T) {
	t.Parallel()
	t.Run("Get Agent - happycase", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := usecasemock.NewMockAgentManageUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// given
		instanceUID := uuid.New()
		agentUsecase.EXPECT().
			GetAgent(mock.Anything, mock.Anything).
			Return(
				//exhaustruct:ignore
				&v1agent.Agent{
					InstanceUID: instanceUID,
				}, nil)
		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/agents/"+instanceUID.String(), nil)
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
		agentUsecase := usecasemock.NewMockAgentManageUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// given
		instanceUID := uuid.New()

		agentUsecase.EXPECT().
			GetAgent(mock.Anything, mock.Anything).
			Return(nil, port.ErrResourceNotExist)
		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/agents/"+instanceUID.String(), nil)
		require.NoError(t, err)
		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusNotFound, recorder.Code)
		assert.JSONEq(t, `{"error":"agent not found"}`, recorder.Body.String())
	})

	t.Run("Get Agent - instanceUID is not uuid returns 400", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := usecasemock.NewMockAgentManageUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/agents/not-a-uuid", nil)
		require.NoError(t, err)
		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("Get Agent - other error returns 500", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := usecasemock.NewMockAgentManageUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// given
		instanceUID := uuid.New()

		agentUsecase.EXPECT().
			GetAgent(mock.Anything, mock.Anything).
			Return(nil, assert.AnError)
		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/agents/"+instanceUID.String(), nil)
		require.NoError(t, err)
		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}

func TestAgentController_UpdateAgentConfig(t *testing.T) {
	t.Parallel()

	t.Run("Update Agent Config - happycase", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := usecasemock.NewMockAgentManageUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// given
		requestBody := `{"targetInstanceUid":"` + uuid.New().String() + `","remoteConfig":{"key":"value"}}`

		agentUsecase.EXPECT().SendCommand(mock.Anything, mock.Anything, mock.Anything).Return(nil)

		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
			fmt.Sprintf("/api/v1/agents/%s/update-agent-config", uuid.New().String()),
			strings.NewReader(requestBody))
		require.NoError(t, err)

		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusCreated, recorder.Code)
	})

	t.Run("Update Agent Config - 400 Bad Request when instanceUID is not uuid", func(t *testing.T) {
		t.Parallel()
		ctrlBase := testutil.NewBase(t).ForController()
		agentManageUsecase := usecasemock.NewMockAgentManageUsecase(t)
		controller := agent.NewController(agentManageUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router
		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
			"/api/v1/agents/not-a-uuid/update-agent-config",
			strings.NewReader(`{"targetInstanceUid":"not-a-uuid","remoteConfig":{"key":"value"}}`))
		require.NoError(t, err)
		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("Update Agent Config - 400 Bad Request when invalid request body", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentManageUsecase := usecasemock.NewMockAgentManageUsecase(t)
		controller := agent.NewController(agentManageUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
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
		agentManageUsecase := usecasemock.NewMockAgentManageUsecase(t)
		controller := agent.NewController(agentManageUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// given
		requestBody := `{"targetInstanceUid":"` + uuid.New().String() + `","remoteConfig":{"key":"value"}}`

		agentManageUsecase.EXPECT().SendCommand(mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError)

		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
			fmt.Sprintf("/api/v1/agents/%s/update-agent-config", uuid.New().String()),
			strings.NewReader(requestBody))
		require.NoError(t, err)

		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}
