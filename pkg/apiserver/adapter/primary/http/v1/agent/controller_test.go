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
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/agent/usecasemock"
	applicationport "github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

var (
	// Ensure Controller implements the Controller interface.
	_ testutil.Controller = (*agent.Controller)(nil)
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
			ListAgents(mock.Anything, "default", mock.Anything).
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
		req, err := http.NewRequestWithContext(
			t.Context(), http.MethodGet, "/api/v1/namespaces/default/agents", nil,
		)
		require.NoError(t, err)

		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "application/json; charset=utf-8", recorder.Header().Get("Content-Type"))
		assert.Equal(t, int64(2), gjson.Get(recorder.Body.String(), "items.#").Int())
		assert.Equal(t, instanceUIDs[0].String(), gjson.Get(recorder.Body.String(), "items.0.metadata.instanceUid").String())
		assert.Equal(t, instanceUIDs[1].String(), gjson.Get(recorder.Body.String(), "items.1.metadata.instanceUid").String())
	})

	t.Run("List Agents - selector parsed into identifying attributes", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := usecasemock.NewMockManageUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// given: repeated selector params are threaded into
		// ListOptions.IdentifyingAttributes as exact key=value pairs.
		agentUsecase.EXPECT().
			ListAgents(mock.Anything, "default", mock.MatchedBy(func(opts *model.ListOptions) bool {
				return opts != nil &&
					opts.IdentifyingAttributes["service.name"] == "otel-collector" &&
					opts.IdentifyingAttributes["service.namespace"] == "prod"
			})).
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
		req, err := http.NewRequestWithContext(
			t.Context(), http.MethodGet,
			"/api/v1/namespaces/default/agents?selector=service.name=otel-collector&selector=service.namespace=prod", nil,
		)
		require.NoError(t, err)

		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("List Agents - selector value may contain commas and equals", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := usecasemock.NewMockManageUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// given: a single selector entry is one key=value pair, split on the first
		// "=" only and never on commas, so the value is preserved verbatim.
		agentUsecase.EXPECT().
			ListAgents(mock.Anything, "default", mock.MatchedBy(func(opts *model.ListOptions) bool {
				return opts != nil &&
					opts.IdentifyingAttributes["service.instance.id"] == "a,b=c"
			})).
			Return(&v1.ListResponse[v1.Agent]{
				APIVersion: "v1",
				Kind:       v1.AgentKind,
				Items:      []v1.Agent{},
				Metadata: v1.ListMeta{
					RemainingItemCount: 0,
					Continue:           "",
				},
			}, nil)

		// when: the value "a,b=c" is URL-encoded so gin hands it back intact.
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(
			t.Context(), http.MethodGet,
			"/api/v1/namespaces/default/agents?selector=service.instance.id%3Da%2Cb%3Dc", nil,
		)
		require.NoError(t, err)

		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("List Agents - malformed selector returns 400", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := usecasemock.NewMockManageUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// when: a selector entry without "=" is rejected before reaching the usecase.
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(
			t.Context(), http.MethodGet,
			"/api/v1/namespaces/default/agents?selector=noequalsign", nil,
		)
		require.NoError(t, err)

		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
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
			ListAgents(mock.Anything, "default", mock.Anything).
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
		req, err := http.NewRequestWithContext(
			t.Context(), http.MethodGet, "/api/v1/namespaces/default/agents", nil,
		)
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
		req, err := http.NewRequestWithContext(
			t.Context(), http.MethodGet, "/api/v1/namespaces/default/agents?limit=invalid", nil,
		)
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
			ListAgents(mock.Anything, "default", mock.Anything).
			Return(nil, assert.AnError)
		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(
			t.Context(), http.MethodGet, "/api/v1/namespaces/default/agents", nil,
		)
		require.NoError(t, err)
		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}

func TestAgentControllerListAgentSelectorThreading(t *testing.T) {
	t.Parallel()

	ctrlBase := testutil.NewBase(t).ForController()
	agentUsecase := usecasemock.NewMockManageUsecase(t)
	controller := agent.NewController(agentUsecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router

	// given: selector and nonIdentifyingSelector map to their own attribute sets.
	agentUsecase.EXPECT().
		ListAgents(mock.Anything, "default", mock.MatchedBy(func(opts *model.ListOptions) bool {
			return opts != nil &&
				opts.IdentifyingAttributes["service.name"] == "otel-collector" &&
				opts.NonIdentifyingAttributes["os.type"] == "linux"
		})).
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
	req, err := http.NewRequestWithContext(
		t.Context(), http.MethodGet,
		"/api/v1/namespaces/default/agents?selector=service.name=otel-collector&nonIdentifyingSelector=os.type=linux",
		nil,
	)
	require.NoError(t, err)

	// then
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
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
			GetAgent(mock.Anything, "default", mock.Anything).
			Return(
				//exhaustruct:ignore
				&v1.Agent{
					Metadata: v1.AgentMetadata{
						InstanceUID: instanceUID,
					},
				}, nil)
		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(
			t.Context(), http.MethodGet,
			"/api/v1/namespaces/default/agents/"+instanceUID.String(), nil,
		)
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
			GetAgent(mock.Anything, "default", mock.Anything).
			Return(nil, port.ErrResourceNotExist)
		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(
			t.Context(), http.MethodGet,
			"/api/v1/namespaces/default/agents/"+instanceUID.String(), nil,
		)
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
		req, err := http.NewRequestWithContext(
			t.Context(), http.MethodGet,
			"/api/v1/namespaces/default/agents/not-a-uuid", nil,
		)
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
			GetAgent(mock.Anything, "default", mock.Anything).
			Return(nil, assert.AnError)
		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(
			t.Context(), http.MethodGet,
			"/api/v1/namespaces/default/agents/"+instanceUID.String(), nil,
		)
		require.NoError(t, err)
		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}

func TestAgentControllerDeleteAgent(t *testing.T) {
	t.Parallel()

	t.Run("Delete Agent - happycase returns 204", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := usecasemock.NewMockManageUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// given
		instanceUID := uuid.New()

		agentUsecase.EXPECT().
			DeleteAgent(mock.Anything, "default", mock.Anything).
			Return(nil)
		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(
			t.Context(), http.MethodDelete,
			"/api/v1/namespaces/default/agents/"+instanceUID.String(), nil,
		)
		require.NoError(t, err)
		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusNoContent, recorder.Code)
		assert.Empty(t, recorder.Body.String())
	})

	t.Run("Delete Agent - connected agent returns 409", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := usecasemock.NewMockManageUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// given
		instanceUID := uuid.New()

		agentUsecase.EXPECT().
			DeleteAgent(mock.Anything, "default", mock.Anything).
			Return(applicationport.ErrAgentConnected)
		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(
			t.Context(), http.MethodDelete,
			"/api/v1/namespaces/default/agents/"+instanceUID.String(), nil,
		)
		require.NoError(t, err)
		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusConflict, recorder.Code)

		body := recorder.Body.String()
		assert.Contains(t, body, "Conflict")
	})

	t.Run("Delete Agent - namespace mismatch returns 404", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := usecasemock.NewMockManageUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// given
		instanceUID := uuid.New()

		agentUsecase.EXPECT().
			DeleteAgent(mock.Anything, "default", mock.Anything).
			Return(applicationport.ErrAgentNamespaceMismatch)
		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(
			t.Context(), http.MethodDelete,
			"/api/v1/namespaces/default/agents/"+instanceUID.String(), nil,
		)
		require.NoError(t, err)
		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusNotFound, recorder.Code)
	})

	t.Run("Delete Agent - not found returns 404", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := usecasemock.NewMockManageUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// given
		instanceUID := uuid.New()

		agentUsecase.EXPECT().
			DeleteAgent(mock.Anything, "default", mock.Anything).
			Return(port.ErrResourceNotExist)
		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(
			t.Context(), http.MethodDelete,
			"/api/v1/namespaces/default/agents/"+instanceUID.String(), nil,
		)
		require.NoError(t, err)
		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusNotFound, recorder.Code)
	})

	t.Run("Delete Agent - invalid uuid returns 400", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := usecasemock.NewMockManageUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(
			t.Context(), http.MethodDelete,
			"/api/v1/namespaces/default/agents/not-a-uuid", nil,
		)
		require.NoError(t, err)
		// then
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
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
			SearchAgents(mock.Anything, "default", "1234", mock.Anything).
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
		req, err := http.NewRequestWithContext(
			t.Context(), http.MethodGet,
			"/api/v1/namespaces/default/agents/search?q=1234", nil,
		)
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
		req, err := http.NewRequestWithContext(
			t.Context(), http.MethodGet,
			"/api/v1/namespaces/default/agents/search", nil,
		)
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
			SearchAgents(mock.Anything, "default", "1234", mock.Anything).
			Return(nil, port.ErrResourceNotExist)

		// when
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(
			t.Context(), http.MethodGet,
			"/api/v1/namespaces/default/agents/search?q=1234", nil,
		)
		require.NoError(t, err)

		router.ServeHTTP(recorder, req)

		// then - ErrResourceNotExist maps to 404
		assert.Equal(t, http.StatusNotFound, recorder.Code)
	})
}
