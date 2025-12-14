package ginutil_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/ginutil"
)

func TestGetQueryInt64(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		query        string
		key          string
		defaultValue int64
		expected     int64
		expectError  bool
	}{
		{
			name:         "valid positive integer",
			query:        "limit=123",
			key:          "limit",
			defaultValue: 0,
			expected:     123,
			expectError:  false,
		},
		{
			name:         "valid zero",
			query:        "limit=0",
			key:          "limit",
			defaultValue: 10,
			expected:     0,
			expectError:  false,
		},
		{
			name:         "valid negative integer",
			query:        "offset=-5",
			key:          "offset",
			defaultValue: 0,
			expected:     -5,
			expectError:  false,
		},
		{
			name:         "empty value returns default",
			query:        "",
			key:          "limit",
			defaultValue: 42,
			expected:     42,
			expectError:  false,
		},
		{
			name:         "key not found returns default",
			query:        "other=value",
			key:          "limit",
			defaultValue: 100,
			expected:     100,
			expectError:  false,
		},
		{
			name:         "invalid format",
			query:        "limit=invalid",
			key:          "limit",
			defaultValue: 0,
			expectError:  true,
		},
		{
			name:         "float value",
			query:        "limit=12.34",
			key:          "limit",
			defaultValue: 0,
			expectError:  true,
		},
		{
			name:         "overflow value",
			query:        "limit=9223372036854775808", // int64 max + 1
			key:          "limit",
			defaultValue: 0,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)

			url := "/test"
			if tt.query != "" {
				url += "?" + tt.query
			}

			ctx.Request = httptest.NewRequest(http.MethodGet, url, nil)

			result, err := ginutil.GetQueryInt64(ctx, tt.key, tt.defaultValue)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get query parameter")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGetErrorTypeURI(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		host        string
		path        string
		method      string
		expectedURI string
	}{
		{
			name:        "simple path",
			host:        "api.example.com",
			path:        "/api/v1/agents",
			method:      http.MethodGet,
			expectedURI: "api.example.com/api/v1/agents",
		},
		{
			name:        "path with parameters",
			host:        "localhost:8080",
			path:        "/api/v1/agents/:id",
			method:      http.MethodGet,
			expectedURI: "localhost:8080/api/v1/agents/:id",
		},
		{
			name:        "root path",
			host:        "example.com",
			path:        "/",
			method:      http.MethodGet,
			expectedURI: "example.com/",
		},
		{
			name:        "empty host",
			host:        "",
			path:        "/api/test",
			method:      http.MethodPost,
			expectedURI: "/api/test",
		},
		{
			name:        "complex path",
			host:        "api.service.local",
			path:        "/api/v1/agentgroups/:name/agents",
			method:      http.MethodGet,
			expectedURI: "api.service.local/api/v1/agentgroups/:name/agents",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := gin.New()

			var actualURI string

			// Add route handler that captures the error type URI
			router.Any(tt.path, func(ctx *gin.Context) {
				actualURI = ginutil.GetErrorTypeURI(ctx)
				ctx.Status(http.StatusOK)
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.Host = tt.host

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedURI, actualURI)
		})
	}
}

func TestGetErrorTypeURI_WithDifferentRoutePatterns(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		routePattern string
		requestPath  string
		host         string
		expectedURI  string
	}{
		{
			name:         "wildcard route",
			routePattern: "/files/*filepath",
			requestPath:  "/files/images/logo.png",
			host:         "cdn.example.com",
			expectedURI:  "cdn.example.com/files/*filepath",
		},
		{
			name:         "multiple parameters",
			routePattern: "/users/:userId/posts/:postId",
			requestPath:  "/users/123/posts/456",
			host:         "blog.example.com",
			expectedURI:  "blog.example.com/users/:userId/posts/:postId",
		},
		{
			name:         "query parameters ignored in URI",
			routePattern: "/search",
			requestPath:  "/search?q=test&limit=10",
			host:         "search.example.com",
			expectedURI:  "search.example.com/search",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := gin.New()

			var actualURI string

			router.GET(tt.routePattern, func(ctx *gin.Context) {
				actualURI = ginutil.GetErrorTypeURI(ctx)
				ctx.Status(http.StatusOK)
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.requestPath, nil)
			req.Host = tt.host

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedURI, actualURI)
		})
	}
}
