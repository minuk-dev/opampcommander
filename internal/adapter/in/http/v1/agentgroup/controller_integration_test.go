package agentgroup_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	agentgroupv1 "github.com/minuk-dev/opampcommander/api/v1/agentgroup"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/agentgroup"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/agentgroup/usecasemock"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestAgentGroupController_ValidationErrorCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		contentType    string
		expectedStatus int
		expectedFields []string
	}{
		{
			name:           "List with negative limit",
			method:         http.MethodGet,
			path:           "/api/v1/agentgroups?limit=-10",
			expectedStatus: http.StatusBadRequest,
			expectedFields: []string{"type", "title", "status", "detail", "instance", "errors"},
		},
		{
			name:           "List with invalid limit format",
			method:         http.MethodGet,
			path:           "/api/v1/agentgroups?limit=abc",
			expectedStatus: http.StatusBadRequest,
			expectedFields: []string{"type", "title", "status", "detail", "instance", "errors"},
		},
		{
			name:           "Get with empty name",
			method:         http.MethodGet,
			path:           "/api/v1/agentgroups/",
			expectedStatus: http.StatusNotFound, // Gin routing returns 404 for missing path param
			expectedFields: []string{},
		},
		{
			name:           "Create with malformed JSON",
			method:         http.MethodPost,
			path:           "/api/v1/agentgroups",
			body:           `{"name":"test", "priority":}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
			expectedFields: []string{"type", "title", "status", "detail", "instance"},
		},
		{
			name:           "Create without content type",
			method:         http.MethodPost,
			path:           "/api/v1/agentgroups",
			body:           `{"name":"test"}`,
			contentType:    "",
			expectedStatus: http.StatusBadRequest,
			expectedFields: []string{"type", "title", "status", "detail", "instance"},
		},
		{
			name:           "Update with invalid JSON",
			method:         http.MethodPut,
			path:           "/api/v1/agentgroups/test-group",
			body:           `invalid json`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
			expectedFields: []string{"type", "title", "status", "detail", "instance"},
		},
		{
			name:           "ListAgentsByAgentGroup with negative limit",
			method:         http.MethodGet,
			path:           "/api/v1/agentgroups/test-group/agents?limit=-5",
			expectedStatus: http.StatusBadRequest,
			expectedFields: []string{"type", "title", "status", "detail", "instance", "errors"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrlBase := testutil.NewBase(t).ForController()
			usecase := usecasemock.NewMockUsecase(t)
			controller := agentgroup.NewController(usecase, ctrlBase.Logger)
			ctrlBase.SetupRouter(controller)
			router := ctrlBase.Router

			var body io.Reader
			if tt.body != "" {
				body = strings.NewReader(tt.body)
			}

			recorder := httptest.NewRecorder()
			req, err := http.NewRequestWithContext(t.Context(), tt.method, tt.path, body)
			require.NoError(t, err)

			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			router.ServeHTTP(recorder, req)
			assert.Equal(t, tt.expectedStatus, recorder.Code)

			if len(tt.expectedFields) > 0 {
				responseBody := recorder.Body.String()
				for _, field := range tt.expectedFields {
					assert.Contains(t, responseBody, field, "Response should contain RFC 9457 field: %s", field)
				}
			}
		})
	}
}

func TestAgentGroupController_ComplexScenarios(t *testing.T) {
	t.Parallel()

	t.Run("Create with all fields", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		usecase := usecasemock.NewMockUsecase(t)
		controller := agentgroup.NewController(usecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		now := time.Now()
		createRequest := agentgroupv1.CreateRequest{
			Name:     "complex-group",
			Priority: 10,
			Attributes: agentgroupv1.Attributes{
				"env":     "production",
				"version": "1.0.0",
			},
			Selector: agentgroupv1.AgentSelector{
				IdentifyingAttributes: map[string]string{
					"service.name": "web-server",
				},
				NonIdentifyingAttributes: map[string]string{
					"region": "us-west-2",
				},
			},
			AgentConfig: &agentgroupv1.AgentConfig{
				Value: `{"sampling_rate": 0.1, "debug": true}`,
			},
		}

		expectedGroup := &agentgroupv1.AgentGroup{
			Name:       createRequest.Name,
			Priority:   createRequest.Priority,
			Attributes: createRequest.Attributes,
			Selector:   createRequest.Selector,
			CreatedAt:  now,
			CreatedBy:  "test-user",
		}

		usecase.EXPECT().
			CreateAgentGroup(mock.Anything, mock.MatchedBy(func(cmd *agentgroup.CreateAgentGroupCommand) bool {
				return cmd.Name == createRequest.Name &&
					cmd.Priority == createRequest.Priority &&
					len(cmd.Attributes) == 2 &&
					cmd.Attributes["env"] == "production"
			})).
			Return(expectedGroup, nil)

		requestBody, err := json.Marshal(createRequest)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/agentgroups", strings.NewReader(string(requestBody)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusCreated, recorder.Code)
		assert.Equal(t, "/api/v1/agentgroups/"+expectedGroup.Name, recorder.Header().Get("Location"))

		var response agentgroupv1.AgentGroup
		err = json.Unmarshal(recorder.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, expectedGroup.Name, response.Name)
		assert.Equal(t, expectedGroup.Priority, response.Priority)
		assert.Equal(t, expectedGroup.Attributes["env"], response.Attributes["env"])
	})

	t.Run("Update with partial fields", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		usecase := usecasemock.NewMockUsecase(t)
		controller := agentgroup.NewController(usecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		groupName := "update-group"
		updateRequest := agentgroupv1.AgentGroup{
			Name:     groupName,
			Priority: 20,
			Attributes: agentgroupv1.Attributes{
				"updated": "true",
			},
		}

		expectedGroup := &agentgroupv1.AgentGroup{
			Name:       groupName,
			Priority:   20,
			Attributes: updateRequest.Attributes,
			CreatedAt:  time.Now(),
			CreatedBy:  "test-user",
		}

		usecase.EXPECT().
			UpdateAgentGroup(mock.Anything, groupName, &updateRequest).
			Return(expectedGroup, nil)

		requestBody, err := json.Marshal(updateRequest)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPut, "/api/v1/agentgroups/"+groupName, strings.NewReader(string(requestBody)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)

		var response agentgroupv1.AgentGroup
		err = json.Unmarshal(recorder.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, expectedGroup.Name, response.Name)
		assert.Equal(t, expectedGroup.Priority, response.Priority)
	})

	t.Run("Delete non-existent group", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		usecase := usecasemock.NewMockUsecase(t)
		controller := agentgroup.NewController(usecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		usecase.EXPECT().
			DeleteAgentGroup(mock.Anything, "non-existent").
			Return(port.ErrResourceNotExist)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, "/api/v1/agentgroups/non-existent", nil)
		require.NoError(t, err)

		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusNotFound, recorder.Code)

		body := recorder.Body.String()
		assert.Contains(t, body, "type")
		assert.Contains(t, body, "title")
		assert.Contains(t, body, "status")
		assert.Contains(t, body, "detail")
		assert.Contains(t, body, "instance")
	})
}

func TestAgentGroupController_ConcurrentOperations(t *testing.T) {
	t.Parallel()

	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := agentgroup.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router

	// Setup mock to handle multiple concurrent List requests
	usecase.EXPECT().
		ListAgentGroups(mock.Anything, mock.Anything).
		Return(&agentgroupv1.ListResponse{
			APIVersion: "v1",
			Kind:       agentgroupv1.AgentGroupKind,
			Items:      []agentgroupv1.AgentGroup{},
			Metadata: v1.ListMeta{
				RemainingItemCount: 0,
				Continue:           "",
			},
		}, nil).
		Times(5) // Expect 5 concurrent calls

	// Run 5 concurrent List requests
	results := make(chan int, 5)

	for i := 0; i < 5; i++ {
		go func() {
			recorder := httptest.NewRecorder()
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/agentgroups", nil)
			require.NoError(t, err)

			router.ServeHTTP(recorder, req)
			results <- recorder.Code
		}()
	}

	// Check all responses
	for i := 0; i < 5; i++ {
		select {
		case code := <-results:
			assert.Equal(t, http.StatusOK, code)
		}
	}
}

func TestAgentGroupController_SpecialCharacters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		groupName    string
		expectedCode int
	}{
		{
			name:         "group name with spaces",
			groupName:    "group with spaces",
			expectedCode: http.StatusOK,
		},
		{
			name:         "group name with special chars",
			groupName:    "group-with_special.chars",
			expectedCode: http.StatusOK,
		},
		{
			name:         "group name with unicode",
			groupName:    "group-测试-グループ",
			expectedCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrlBase := testutil.NewBase(t).ForController()
			usecase := usecasemock.NewMockUsecase(t)
			controller := agentgroup.NewController(usecase, ctrlBase.Logger)
			ctrlBase.SetupRouter(controller)
			router := ctrlBase.Router

			expectedGroup := &agentgroupv1.AgentGroup{
				Name:     tt.groupName,
				Priority: 1,
			}

			usecase.EXPECT().
				GetAgentGroup(mock.Anything, tt.groupName).
				Return(expectedGroup, nil)

			recorder := httptest.NewRecorder()
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/agentgroups/"+tt.groupName, nil)
			require.NoError(t, err)

			router.ServeHTTP(recorder, req)
			assert.Equal(t, tt.expectedCode, recorder.Code)

			if tt.expectedCode == http.StatusOK {
				responseBody := recorder.Body.String()
				assert.Contains(t, responseBody, tt.groupName)
			}
		})
	}
}

func TestAgentGroupController_LargePayloads(t *testing.T) {
	t.Parallel()

	t.Run("Create with large attributes", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		usecase := usecasemock.NewMockUsecase(t)
		controller := agentgroup.NewController(usecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		// Create large attributes map
		largeAttributes := make(agentgroupv1.Attributes)
		for i := 0; i < 100; i++ {
			key := fmt.Sprintf("key_%d", i)
			value := strings.Repeat("value", 100) // 500 character value
			largeAttributes[key] = value
		}

		createRequest := agentgroupv1.CreateRequest{
			Name:       "large-group",
			Priority:   1,
			Attributes: largeAttributes,
		}

		expectedGroup := &agentgroupv1.AgentGroup{
			Name:       createRequest.Name,
			Priority:   createRequest.Priority,
			Attributes: createRequest.Attributes,
			CreatedAt:  time.Now(),
		}

		usecase.EXPECT().
			CreateAgentGroup(mock.Anything, mock.Anything).
			Return(expectedGroup, nil)

		requestBody, err := json.Marshal(createRequest)
		require.NoError(t, err)
		require.Greater(t, len(requestBody), 50000) // Ensure payload is actually large

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/agentgroups", strings.NewReader(string(requestBody)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusCreated, recorder.Code)

		// Verify response contains some of the large data
		responseBody := recorder.Body.String()
		assert.Contains(t, responseBody, "large-group")
		assert.Contains(t, responseBody, "key_50") // Check middle key exists
	})
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func TestAgentGroupController_MimeTypeHandling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		contentType string
		expectError bool
	}{
		{
			name:        "application/json",
			contentType: "application/json",
			expectError: false,
		},
		{
			name:        "application/json with charset",
			contentType: "application/json; charset=utf-8",
			expectError: false,
		},
		{
			name:        "text/plain",
			contentType: "text/plain",
			expectError: true,
		},
		{
			name:        "application/xml",
			contentType: "application/xml",
			expectError: true,
		},
		{
			name:        "empty content type",
			contentType: "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrlBase := testutil.NewBase(t).ForController()
			usecase := usecasemock.NewMockUsecase(t)
			controller := agentgroup.NewController(usecase, ctrlBase.Logger)
			ctrlBase.SetupRouter(controller)
			router := ctrlBase.Router

			if !tt.expectError {
				usecase.EXPECT().
					CreateAgentGroup(mock.Anything, mock.Anything).
					Return(&agentgroupv1.AgentGroup{
						Name: "test-group",
					}, nil)
			}

			createRequest := agentgroupv1.CreateRequest{
				Name:     "test-group",
				Priority: 1,
			}

			requestBody, err := json.Marshal(createRequest)
			require.NoError(t, err)

			recorder := httptest.NewRecorder()
			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/agentgroups", strings.NewReader(string(requestBody)))
			require.NoError(t, err)

			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			router.ServeHTTP(recorder, req)

			if tt.expectError {
				assert.Equal(t, http.StatusBadRequest, recorder.Code)
				body := recorder.Body.String()
				assert.Contains(t, body, "type")
				assert.Contains(t, body, "title")
			} else {
				assert.Equal(t, http.StatusCreated, recorder.Code)
			}
		})
	}
}