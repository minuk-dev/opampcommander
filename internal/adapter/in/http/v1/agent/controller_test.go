package agent_test

import (
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
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/agent"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/agent/usecasemock"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/module/helper"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

var (
	// Ensure Controller implements ginutil.Controller interface.
	_ helper.Controller = (*agent.Controller)(nil)
)

func TestAgentControllerListAgent(t *testing.T) {
	t.Parallel()

	t.Run("List Agents - happycase", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := usecasemock.NewMockManageUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router
		// given
		instanceUIDs := []uuid.UUID{uuid.New(), uuid.New()}
		//exhaustruct:ignore
		agents := []v1.Agent{
			{
				Metadata: v1.AgentMetadata{
					InstanceUID: instanceUIDs[0],
				},
			},
			{
				Metadata: v1.AgentMetadata{
					InstanceUID: instanceUIDs[1],
				},
			},
		}
		agentUsecase.EXPECT().
			ListAgents(mock.Anything, mock.Anything).
			Return(&v1.ListResponse[v1.Agent]{
				APIVersion: "v1",
				Kind:       v1.AgentKind,
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
		assert.Equal(t, instanceUIDs[0].String(), gjson.Get(recorder.Body.String(), "items.0.metadata.instanceUid").String())
		assert.Equal(t, instanceUIDs[1].String(), gjson.Get(recorder.Body.String(), "items.1.metadata.instanceUid").String())
	})

	t.Run("List Agents - empty returns 200, empty", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := usecasemock.NewMockManageUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// given
		agentUsecase.EXPECT().
			ListAgents(mock.Anything, mock.Anything).
			Return(&v1.ListResponse[v1.Agent]{
				APIVersion: "v1",
				Kind:       v1.AgentKind,
				Items:      []v1.Agent{},
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
		agentUsecase := usecasemock.NewMockManageUsecase(t)
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

		// Check RFC 9457 structure
		body := recorder.Body.String()
		assert.Contains(t, body, "type")
		assert.Contains(t, body, "title")
		assert.Contains(t, body, "status")
		assert.Contains(t, body, "detail")
		assert.Contains(t, body, "instance")
		assert.Contains(t, body, "errors")

		// Check specific error information
		assert.Contains(t, body, "invalid format")
		assert.Contains(t, body, "query.limit")
		assert.Contains(t, body, "invalid")
	})

	t.Run("List Agents - any error returns 500", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := usecasemock.NewMockManageUsecase(t)
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
		agentUsecase := usecasemock.NewMockManageUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// given
		instanceUID := uuid.New()
		agentUsecase.EXPECT().
			GetAgent(mock.Anything, mock.Anything).
			Return(
				//exhaustruct:ignore
				&v1.Agent{
					Metadata: v1.AgentMetadata{
						InstanceUID: instanceUID,
					},
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
		assert.Equal(t, instanceUID.String(), gjson.Get(recorder.Body.String(), "metadata.instanceUid").String())
	})

	t.Run("Get Agent - not found error returns 404", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := usecasemock.NewMockManageUsecase(t)
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

		// Check RFC 9457 structure
		body := recorder.Body.String()
		assert.Contains(t, body, "type")
		assert.Contains(t, body, "title")
		assert.Contains(t, body, "status")
		assert.Contains(t, body, "detail")
		assert.Contains(t, body, "instance")
	})

	t.Run("Get Agent - instanceUID is not uuid returns 400", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := usecasemock.NewMockManageUsecase(t)
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

		// Check RFC 9457 structure
		body := recorder.Body.String()
		assert.Contains(t, body, "type")
		assert.Contains(t, body, "title")
		assert.Contains(t, body, "status")
		assert.Contains(t, body, "detail")
		assert.Contains(t, body, "instance")
		assert.Contains(t, body, "errors")

		// Check specific error information
		assert.Contains(t, body, "invalid format")
		assert.Contains(t, body, "path.id")
		assert.Contains(t, body, "not-a-uuid")
	})

	t.Run("Get Agent - other error returns 500", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := usecasemock.NewMockManageUsecase(t)
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

func TestAgentControllerSearchAgent(t *testing.T) {
	t.Parallel()

	t.Run("Search Agents - happy case", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := usecasemock.NewMockManageUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// given
		instanceUID := uuid.MustParse("12345678-1234-1234-1234-123456789012")
		//exhaustruct:ignore
		agents := []v1.Agent{
			{
				Metadata: v1.AgentMetadata{
					InstanceUID: instanceUID,
				},
			},
		}
		agentUsecase.EXPECT().
			SearchAgents(mock.Anything, "1234", mock.Anything).
			Return(&v1.ListResponse[v1.Agent]{
				APIVersion: "v1",
				Kind:       v1.AgentKind,
				Items:      agents,
				Metadata: v1.ListMeta{
					RemainingItemCount: 0,
					Continue:           "",
				},
			}, nil)

		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/agents/search?q=1234", nil)
		require.NoError(t, err)

		router.ServeHTTP(recorder, req)

		// then
		assert.Equal(t, http.StatusOK, recorder.Code)

		result := gjson.Parse(recorder.Body.String())
		assert.Equal(t, "v1", result.Get("apiVersion").String())
		assert.Equal(t, v1.AgentKind, result.Get("kind").String())
		assert.Equal(t, int64(1), result.Get("items.#").Int())
		assert.Equal(t, instanceUID.String(), result.Get("items.0.metadata.instanceUid").String())
	})

	t.Run("Search Agents - missing query parameter", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := usecasemock.NewMockManageUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/agents/search", nil)
		require.NoError(t, err)

		router.ServeHTTP(recorder, req)

		// then - Should return 400 for missing required query parameter
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("Search Agents - usecase error returns 404", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := usecasemock.NewMockManageUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// given
		agentUsecase.EXPECT().
			SearchAgents(mock.Anything, "1234", mock.Anything).
			Return(nil, port.ErrResourceNotExist)

		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/agents/search?q=1234", nil)
		require.NoError(t, err)

		router.ServeHTTP(recorder, req)

		// then - ErrResourceNotExist maps to 404
		assert.Equal(t, http.StatusNotFound, recorder.Code)
	})
}
