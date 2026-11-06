package security_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/inmemory"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/security"
)

func TestAuthJWTMiddleware_ProblemResponse(t *testing.T) {
	t.Parallel()

	tests := []struct{ name, authorization string }{
		{"missing token", ""},
		{"invalid token", "Bearer invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := newBasicAuthService(t, "", inmemory.NewUserRepository())
			router := gin.New()
			router.Use(security.NewAuthJWTMiddleware(service))

			called := false

			router.GET("/api/v1/users", func(ctx *gin.Context) {
				called = true

				ctx.Status(http.StatusOK)
			})

			recorder := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/users", nil)
			req.Header.Set("Authorization", tt.authorization)
			router.ServeHTTP(recorder, req)

			assert.False(t, called)
			assert.Equal(t, http.StatusUnauthorized, recorder.Code)
			assert.Equal(t, "application/problem+json", recorder.Header().Get("Content-Type"))
			assert.JSONEq(t, `{
				"type": "about:blank", "title": "Unauthorized", "status": 401,
				"detail": "unauthorized", "instance": "/api/v1/users"
			}`, recorder.Body.String())
		})
	}
}
