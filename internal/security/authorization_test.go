package security_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	userport "github.com/minuk-dev/opampcommander/internal/domain/user/port"
	"github.com/minuk-dev/opampcommander/internal/security"
)

func TestMain(m *testing.M) { goleak.VerifyTestMain(m) }

const adminEmail = "admin@example.com"

// buildAuthzRouter creates a gin engine that pre-sets the given user in context
// (simulating JWT middleware) then runs NewAuthorizationMiddleware.
// Each of the me-family routes returns 200 if the middleware allows the request through.
func buildAuthzRouter(user *security.User) *gin.Engine {
	router := gin.New()

	router.Use(func(ctx *gin.Context) {
		security.SetUser(ctx, user)
		ctx.Next()
	})

	var (
		rbac  userport.RBACUsecase
		users userport.UserUsecase
	)

	router.Use(security.NewAuthorizationMiddleware(rbac, users, adminEmail, slog.Default()))

	router.GET("/api/v1/users/me", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	return router
}

func TestAuthorizationMiddleware_UsersMe_RequiresAuthentication(t *testing.T) {
	t.Parallel()

	t.Run("rejects anonymous user with 401", func(t *testing.T) {
		t.Parallel()

		router := buildAuthzRouter(security.NewAnonymousUser())

		w := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/users/me", nil)
		require.NoError(t, err)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("allows authenticated non-admin user", func(t *testing.T) {
		t.Parallel()

		email := "user@example.com"
		router := buildAuthzRouter(&security.User{Authenticated: true, Email: &email})

		w := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/users/me", nil)
		require.NoError(t, err)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("allows admin user", func(t *testing.T) {
		t.Parallel()

		email := adminEmail
		router := buildAuthzRouter(&security.User{Authenticated: true, Email: &email})

		w := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/users/me", nil)
		require.NoError(t, err)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
