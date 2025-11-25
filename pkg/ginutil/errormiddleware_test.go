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

// TestErrorResponseStructure tests that the error response follows RFC 9457 structure.
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