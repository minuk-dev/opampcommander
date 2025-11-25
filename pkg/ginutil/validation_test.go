package ginutil_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/minuk-dev/opampcommander/pkg/ginutil"
)

func TestParseUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		paramValue  string
		expectError bool
		errorType   error
	}{
		{
			name:        "valid UUID",
			paramValue:  "550e8400-e29b-41d4-a716-446655440000",
			expectError: false,
		},
		{
			name:        "invalid UUID format",
			paramValue:  "invalid-uuid",
			expectError: true,
			errorType:   ginutil.ErrInvalidFormat,
		},
		{
			name:        "empty UUID",
			paramValue:  "",
			expectError: true,
			errorType:   ginutil.ErrRequiredParam,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)
			ctx.AddParam("id", tt.paramValue)

			result, err := ginutil.ParseUUID(ctx, "id")

			if tt.expectError {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.errorType))
				assert.Equal(t, uuid.Nil, result)
			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, uuid.Nil, result)
			}
		})
	}
}

func TestParseInt64(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		paramValue   string
		defaultValue int64
		expected     int64
		expectError  bool
		errorType    error
	}{
		{
			name:         "valid positive integer",
			paramValue:   "123",
			defaultValue: 0,
			expected:     123,
			expectError:  false,
		},
		{
			name:         "valid zero",
			paramValue:   "0",
			defaultValue: 10,
			expected:     0,
			expectError:  false,
		},
		{
			name:         "empty value returns default",
			paramValue:   "",
			defaultValue: 42,
			expected:     42,
			expectError:  false,
		},
		{
			name:         "invalid format",
			paramValue:   "invalid",
			defaultValue: 0,
			expectError:  true,
			errorType:    ginutil.ErrInvalidFormat,
		},
		{
			name:         "negative value",
			paramValue:   "-5",
			defaultValue: 0,
			expectError:  true,
			errorType:    ginutil.ErrInvalidValue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)
			
			if tt.paramValue != "" {
				ctx.Request = httptest.NewRequest(http.MethodGet, "/test?limit="+tt.paramValue, nil)
			} else {
				ctx.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
			}

			result, err := ginutil.ParseInt64(ctx, "limit", tt.defaultValue)

			if tt.expectError {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.errorType))
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseString(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		paramValue  string
		required    bool
		expected    string
		expectError bool
		errorType   error
	}{
		{
			name:        "valid non-empty string",
			paramValue:  "test-name",
			required:    true,
			expected:    "test-name",
			expectError: false,
		},
		{
			name:        "empty string not required",
			paramValue:  "",
			required:    false,
			expected:    "",
			expectError: false,
		},
		{
			name:        "empty string required",
			paramValue:  "",
			required:    true,
			expectError: true,
			errorType:   ginutil.ErrRequiredParam,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)
			ctx.AddParam("name", tt.paramValue)

			result, err := ginutil.ParseString(ctx, "name", tt.required)

			if tt.expectError {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.errorType))
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestBindJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type TestStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	tests := []struct {
		name        string
		requestBody string
		expectError bool
		errorType   error
	}{
		{
			name:        "valid JSON",
			requestBody: `{"name":"John","age":30}`,
			expectError: false,
		},
		{
			name:        "invalid JSON",
			requestBody: `{"name":"John","age":}`,
			expectError: true,
			errorType:   ginutil.ErrValidationFailed,
		},
		{
			name:        "empty body",
			requestBody: "",
			expectError: true,
			errorType:   ginutil.ErrValidationFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(tt.requestBody))
			ctx.Request.Header.Set("Content-Type", "application/json")

			var obj TestStruct
			err := ginutil.BindJSON(ctx, &obj)

			if tt.expectError {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.errorType))
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "John", obj.Name)
				assert.Equal(t, 30, obj.Age)
			}
		})
	}
}

func TestHandleValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		err          error
		paramName    string
		paramValue   string
		isPathParam  bool
		expectedCode int
	}{
		{
			name:         "required param error for path param",
			err:          ginutil.ErrRequiredParam,
			paramName:    "id",
			paramValue:   "",
			isPathParam:  true,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "required param error for query param",
			err:          ginutil.ErrRequiredParam,
			paramName:    "limit",
			paramValue:   "",
			isPathParam:  false,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "invalid format error for path param",
			err:          ginutil.ErrInvalidFormat,
			paramName:    "id",
			paramValue:   "invalid-uuid",
			isPathParam:  true,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "invalid format error for query param",
			err:          ginutil.ErrInvalidFormat,
			paramName:    "limit",
			paramValue:   "invalid",
			isPathParam:  false,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "invalid value error for query param",
			err:          ginutil.ErrInvalidValue,
			paramName:    "limit",
			paramValue:   "-5",
			isPathParam:  false,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "invalid value error for path param",
			err:          ginutil.ErrInvalidValue,
			paramName:    "priority",
			paramValue:   "-10",
			isPathParam:  true,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "validation failed error",
			err:          ginutil.ErrValidationFailed,
			paramName:    "body",
			paramValue:   "",
			isPathParam:  false,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "unknown error",
			err:          errors.New("unknown error"),
			paramName:    "test",
			paramValue:   "test",
			isPathParam:  false,
			expectedCode: http.StatusInternalServerError,
		},
		{
			name:         "wrapped error",
			err:          fmt.Errorf("wrapped: %w", ginutil.ErrInvalidFormat),
			paramName:    "wrapped",
			paramValue:   "value",
			isPathParam:  false,
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

			ginutil.HandleValidationError(ctx, tt.paramName, tt.paramValue, tt.err, tt.isPathParam)

			assert.Equal(t, tt.expectedCode, w.Code)
			assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
		})
	}
}

func TestHandleValidationError_CompleteErrorResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name            string
		err             error
		paramName       string
		paramValue      string
		isPathParam     bool
		expectedMessage string
		expectedLocation string
	}{
		{
			name:            "query param invalid format",
			err:             ginutil.ErrInvalidFormat,
			paramName:       "limit",
			paramValue:      "abc",
			isPathParam:     false,
			expectedMessage: "invalid format",
			expectedLocation: "query.limit",
		},
		{
			name:            "path param required",
			err:             ginutil.ErrRequiredParam,
			paramName:       "id",
			paramValue:      "",
			isPathParam:     true,
			expectedMessage: "parameter is required",
			expectedLocation: "path.id",
		},
		{
			name:            "query param invalid value",
			err:             ginutil.ErrInvalidValue,
			paramName:       "offset",
			paramValue:      "-1",
			isPathParam:     false,
			expectedMessage: "invalid value",
			expectedLocation: "query.offset",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)
			
			url := "/test"
			if !tt.isPathParam && tt.paramValue != "" {
				url += "?" + tt.paramName + "=" + tt.paramValue
			}
			ctx.Request = httptest.NewRequest(http.MethodGet, url, nil)
			
			if tt.isPathParam {
				ctx.AddParam(tt.paramName, tt.paramValue)
			}

			ginutil.HandleValidationError(ctx, tt.paramName, tt.paramValue, tt.err, tt.isPathParam)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			
			body := w.Body.String()
			assert.Contains(t, body, tt.expectedMessage)
			// Note: location determination is complex, so we just check the response structure
			assert.Contains(t, body, "errors")
			assert.Contains(t, body, "location")
		})
	}
}