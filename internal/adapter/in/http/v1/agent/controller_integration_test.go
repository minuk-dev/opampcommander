//nolint:dupl
package agent_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	v1agent "github.com/minuk-dev/opampcommander/api/v1/agent"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/agent"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/agent/usecasemock"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestAgentController_ValidationErrorCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		expectedFields []string
	}{
		{
			name:           "List with negative limit",
			method:         http.MethodGet,
			path:           "/api/v1/agents?limit=-10",
			expectedStatus: http.StatusBadRequest,
			expectedFields: []string{"type", "title", "status", "detail", "instance", "errors"},
		},
		{
			name:           "List with invalid limit format",
			method:         http.MethodGet,
			path:           "/api/v1/agents?limit=abc",
			expectedStatus: http.StatusBadRequest,
			expectedFields: []string{"type", "title", "status", "detail", "instance", "errors"},
		},
		{
			name:           "List with float limit",
			method:         http.MethodGet,
			path:           "/api/v1/agents?limit=12.5",
			expectedStatus: http.StatusBadRequest,
			expectedFields: []string{"type", "title", "status", "detail", "instance", "errors"},
		},
		{
			name:           "Get with empty UUID",
			method:         http.MethodGet,
			path:           "/api/v1/agents/",
			expectedStatus: http.StatusNotFound, // Gin routing returns 404 for missing path param
			expectedFields: []string{},
		},
		{
			name:           "Get with malformed UUID",
			method:         http.MethodGet,
			path:           "/api/v1/agents/not-a-uuid",
			expectedStatus: http.StatusBadRequest,
			expectedFields: []string{"type", "title", "status", "detail", "instance", "errors"},
		},
		{
			name:           "Get with partial UUID",
			method:         http.MethodGet,
			path:           "/api/v1/agents/123e4567-e89b-12d3-a456",
			expectedStatus: http.StatusBadRequest,
			expectedFields: []string{"type", "title", "status", "detail", "instance", "errors"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrlBase := testutil.NewBase(t).ForController()
			agentUsecase := usecasemock.NewMockManageUsecase(t)
			controller := agent.NewController(agentUsecase, ctrlBase.Logger)
			ctrlBase.SetupRouter(controller)
			router := ctrlBase.Router

			recorder := httptest.NewRecorder()
			req, err := http.NewRequestWithContext(t.Context(), tt.method, tt.path, nil)
			require.NoError(t, err)

			router.ServeHTTP(recorder, req)
			assert.Equal(t, tt.expectedStatus, recorder.Code)

			if len(tt.expectedFields) > 0 {
				body := recorder.Body.String()
				for _, field := range tt.expectedFields {
					assert.Contains(t, body, field, "Response should contain RFC 9457 field: %s", field)
				}
			}
		})
	}
}

func TestAgentController_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("List with very large limit", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := usecasemock.NewMockManageUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// Mock expects to be called with large limit
		agentUsecase.EXPECT().
			ListAgents(mock.Anything, mock.MatchedBy(func(opts *model.ListOptions) bool {
				return opts.Limit == 9223372036854775807 // Max int64
			})).
			Return(&v1agent.ListResponse{
				APIVersion: "v1",
				Kind:       v1agent.AgentKind,
				Items:      []v1agent.Agent{},
				Metadata: v1.ListMeta{
					RemainingItemCount: 0,
					Continue:           "",
				},
			}, nil)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/agents?limit=9223372036854775807", nil)
		require.NoError(t, err)

		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("List with zero limit", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := usecasemock.NewMockManageUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		agentUsecase.EXPECT().
			ListAgents(mock.Anything, mock.MatchedBy(func(opts *model.ListOptions) bool {
				return opts.Limit == 0
			})).
			Return(&v1agent.ListResponse{
				APIVersion: "v1",
				Kind:       v1agent.AgentKind,
				Items:      []v1agent.Agent{},
				Metadata: v1.ListMeta{
					RemainingItemCount: 0,
					Continue:           "",
				},
			}, nil)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/agents?limit=0", nil)
		require.NoError(t, err)

		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("Get with valid UUID format but different case", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := usecasemock.NewMockManageUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		uppercaseUUID := "A1B2C3D4-E5F6-7890-ABCD-EF1234567890"
		expectedUUID, err := uuid.Parse(uppercaseUUID)
		require.NoError(t, err)

		agentUsecase.EXPECT().
			GetAgent(mock.Anything, expectedUUID).
			Return(&v1agent.Agent{
				Metadata: v1agent.Metadata{
					InstanceUID: expectedUUID,
				},
			}, nil)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/agents/"+uppercaseUUID, nil)
		require.NoError(t, err)

		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)

		// Verify the UUID is properly normalized
		responseUUID := gjson.Get(recorder.Body.String(), "metadata.instanceUid").String()
		assert.Equal(t, expectedUUID.String(), responseUUID)
	})
}

func TestAgentController_ConcurrentRequests(t *testing.T) {
	t.Parallel()

	ctrlBase := testutil.NewBase(t).ForController()
	agentUsecase := usecasemock.NewMockManageUsecase(t)
	controller := agent.NewController(agentUsecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router

	// Setup mock to handle multiple concurrent requests
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
		}, nil).
		Times(10) // Expect 10 concurrent calls

	// Run 10 concurrent requests
	responses := make(chan int, 10)

	for range 10 {
		go func() {
			recorder := httptest.NewRecorder()
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/agents", nil)
			assert.NoError(t, err)

			router.ServeHTTP(recorder, req)

			responses <- recorder.Code
		}()
	}

	// Check all responses
	for range 10 {
		code := <-responses
		assert.Equal(t, http.StatusOK, code)
	}
}

func TestAgentController_ResponseFormat(t *testing.T) {
	t.Parallel()

	t.Run("List response has correct structure", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := usecasemock.NewMockManageUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		instanceUID := uuid.New()
		agentUsecase.EXPECT().
			ListAgents(mock.Anything, mock.Anything).
			Return(&v1agent.ListResponse{
				APIVersion: "v1",
				Kind:       v1agent.AgentKind,
				Items: []v1agent.Agent{
					{
						Metadata: v1agent.Metadata{
							InstanceUID: instanceUID,
						},
					},
				},
				Metadata: v1.ListMeta{
					RemainingItemCount: 5,
					Continue:           "next-token",
				},
			}, nil)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/agents", nil)
		require.NoError(t, err)

		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)

		var response v1agent.ListResponse

		err = json.Unmarshal(recorder.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "v1", response.APIVersion)
		assert.Equal(t, v1agent.AgentKind, response.Kind)
		assert.Len(t, response.Items, 1)
		assert.Equal(t, instanceUID, response.Items[0].Metadata.InstanceUID)
		assert.Equal(t, int64(5), response.Metadata.RemainingItemCount)
		assert.Equal(t, "next-token", response.Metadata.Continue)
	})

	t.Run("Get response has correct structure", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		agentUsecase := usecasemock.NewMockManageUsecase(t)
		controller := agent.NewController(agentUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		instanceUID := uuid.New()
		agentUsecase.EXPECT().
			GetAgent(mock.Anything, instanceUID).
			Return(&v1agent.Agent{
				Metadata: v1agent.Metadata{
					InstanceUID: instanceUID,
					Description: v1agent.Description{
						IdentifyingAttributes: map[string]string{
							"service.name": "test-service",
						},
					},
				},
			}, nil)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/agents/"+instanceUID.String(), nil)
		require.NoError(t, err)

		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)

		var response v1agent.Agent

		err = json.Unmarshal(recorder.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, instanceUID, response.Metadata.InstanceUID)
		assert.Equal(t, "test-service", response.Metadata.Description.IdentifyingAttributes["service.name"])
	})
}
