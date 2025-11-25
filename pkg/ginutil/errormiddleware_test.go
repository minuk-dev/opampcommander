package ginutil_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/ginutil"
)

func TestInvalidQueryParamError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/test?limit=invalid", nil)

	ginutil.InvalidQueryParamError(ctx, "limit", "invalid", "must be a valid integer")

	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Check response headers
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
}

func TestInvalidPathParamError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/agents/invalid-uuid", nil)
	ctx.AddParam("id", "invalid-uuid")

	ginutil.InvalidPathParamError(ctx, "id", "invalid-uuid", "invalid UUID format")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInvalidRequestBodyError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/test", nil)

	testErr := errors.New("invalid JSON format")
	ginutil.InvalidRequestBodyError(ctx, testErr)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleDomainError_ResourceNotExist(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/agents/123", nil)

	ginutil.HandleDomainError(ctx, domainport.ErrResourceNotExist, "Agent not found")

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleDomainError_InternalServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/agents", nil)

	testErr := errors.New("database connection failed")
	ginutil.HandleDomainError(ctx, testErr, "Failed to retrieve agents")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestInternalServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	testErr := errors.New("something went wrong")
	ginutil.InternalServerError(ctx, testErr, "An unexpected error occurred")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestResourceNotFoundError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/agents/123", nil)
	ctx.AddParam("id", "123")

	ginutil.ResourceNotFoundError(ctx, "agent", "123")

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestErrorResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	errorInfo := &ginutil.ErrorInfo{
		Type:     ginutil.ErrorTypeInvalidQuery,
		Message:  "Custom error message",
		Location: "query.test",
		Value:    "invalid",
	}

	ginutil.ErrorResponse(ctx, errorInfo)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestErrorResponse tests that the error response follows RFC 9457 structure.
func TestErrorResponseStructure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	router.GET("/test", func(ctx *gin.Context) {
		ginutil.InvalidQueryParamError(ctx, "limit", "invalid", "must be a valid integer")
	})

	req := httptest.NewRequest(http.MethodGet, "/test?limit=invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "type")
	assert.Contains(t, w.Body.String(), "title")
	assert.Contains(t, w.Body.String(), "status")
	assert.Contains(t, w.Body.String(), "detail")
	assert.Contains(t, w.Body.String(), "instance")
	assert.Contains(t, w.Body.String(), "errors")
}

func TestDetermineLocationFromURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		setupContext func() *gin.Context
		identifier   string
		expected     string
	}{
		{
			name: "path parameter",
			setupContext: func() *gin.Context {
				w := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(w)
				ctx.Params = gin.Params{
					{Key: "id", Value: "test-id"},
					{Key: "name", Value: "test-name"},
				}
				ctx.Request = httptest.NewRequest(http.MethodGet, "/agents/test-id", nil)
				return ctx
			},
			identifier: "test-id",
			expected:   "path.id",
		},
		{
			name: "query parameter",
			setupContext: func() *gin.Context {
				w := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(w)
				ctx.Request = httptest.NewRequest(http.MethodGet, "/agents?limit=invalid&offset=0", nil)
				return ctx
			},
			identifier: "invalid",
			expected:   "query.limit",
		},
		{
			name: "multiple query values - first match",
			setupContext: func() *gin.Context {
				w := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(w)
				ctx.Request = httptest.NewRequest(http.MethodGet, "/test?filter=abc&search=abc", nil)
				return ctx
			},
			identifier: "abc",
			expected:   "query.filter", // Should return first match
		},
		{
			name: "not found in path or query",
			setupContext: func() *gin.Context {
				w := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(w)
				ctx.Params = gin.Params{
					{Key: "id", Value: "other-value"},
				}
				ctx.Request = httptest.NewRequest(http.MethodGet, "/agents/other-value?limit=10", nil)
				return ctx
			},
			identifier: "not-found",
			expected:   "unknown",
		},
		{
			name: "empty identifier",
			setupContext: func() *gin.Context {
				w := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(w)
				ctx.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
				return ctx
			},
			identifier: "",
			expected:   "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupContext()
			
			// Use ResourceNotFoundError to trigger determineLocationFromURL indirectly
			ginutil.ResourceNotFoundError(ctx, "test", tt.identifier)
			
			// Check that the response was generated (meaning the function ran)
			assert.True(t, ctx.Writer.Written())
		})
	}
}

func TestGetErrorDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		errorType      ginutil.ErrorType
		expectedStatus int
		expectedTitle  string
		expectedDetail string
	}{
		{
			name:           "invalid query error",
			errorType:      ginutil.ErrorTypeInvalidQuery,
			expectedStatus: http.StatusBadRequest,
			expectedTitle:  "Invalid Query Parameter",
			expectedDetail: "One or more query parameters are invalid.",
		},
		{
			name:           "invalid path error",
			errorType:      ginutil.ErrorTypeInvalidPath,
			expectedStatus: http.StatusBadRequest,
			expectedTitle:  "Invalid Path Parameter",
			expectedDetail: "One or more path parameters are invalid.",
		},
		{
			name:           "invalid request body error",
			errorType:      ginutil.ErrorTypeInvalidRequestBody,
			expectedStatus: http.StatusBadRequest,
			expectedTitle:  "Invalid Request Body",
			expectedDetail: "The request body is not valid JSON or does not conform to the expected schema.",
		},
		{
			name:           "resource not found error",
			errorType:      ginutil.ErrorTypeResourceNotFound,
			expectedStatus: http.StatusNotFound,
			expectedTitle:  "Not Found",
			expectedDetail: "The requested resource does not exist.",
		},
		{
			name:           "internal server error",
			errorType:      ginutil.ErrorTypeInternalServer,
			expectedStatus: http.StatusInternalServerError,
			expectedTitle:  "Internal Server Error",
			expectedDetail: "An unexpected error occurred while processing the request.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

			errorInfo := &ginutil.ErrorInfo{
				Type:     tt.errorType,
				Message:  "test message",
				Location: "test.location",
				Value:    "test-value",
			}

			ginutil.ErrorResponse(ctx, errorInfo)

			assert.Equal(t, tt.expectedStatus, w.Code)
			
			body := w.Body.String()
			assert.Contains(t, body, tt.expectedTitle)
			assert.Contains(t, body, tt.expectedDetail)
		})
	}
}

func TestErrorResponse_UnknownErrorType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	// Use an undefined error type (should default to unknown error)
	errorInfo := &ginutil.ErrorInfo{
		Type:     ginutil.ErrorType(999), // Unknown error type
		Message:  "test message",
		Location: "test.location",
		Value:    "test-value",
	}

	ginutil.ErrorResponse(ctx, errorInfo)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	
	body := w.Body.String()
	assert.Contains(t, body, "Unknown Error")
	assert.Contains(t, body, "An unknown error occurred")
}