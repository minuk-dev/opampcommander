package ginutil_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/ginutil"
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

			ctx.Request = httptest.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)

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
