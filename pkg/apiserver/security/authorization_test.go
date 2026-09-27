package security_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	usermodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user"
	userport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/security"
)

func TestMain(m *testing.M) {
	// memguard starts a single process-lifetime daemon (the Coffer rekeying goroutine) the
	// first time an enclave is created; it never exits by design, so it is not a leak.
	goleak.VerifyTestMain(m,
		goleak.IgnoreTopFunction("github.com/awnumar/memguard/core.NewCoffer.func1"),
	)
}

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

type namespaceRBAC struct {
	userport.RBACUsecase

	check func(string, string, string) bool
}

func (r namespaceRBAC) CheckPermission(
	_ context.Context, _ uuid.UUID, namespace, resource, action string,
) (bool, error) {
	return r.check(namespace, resource, action), nil
}

type namespaceUsers struct{ userport.UserUsecase }

func (namespaceUsers) GetUserByEmail(_ context.Context, email string) (*usermodel.User, error) {
	return usermodel.NewUser(email, email), nil
}

func TestAuthorizationMiddleware_NamespacePermissions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, method, path, scope, action string
		allowed                           bool
	}{
		{"list requires cluster scope", http.MethodGet, "/api/v1/namespaces", "*", "LIST", false},
		{"create requires cluster scope", http.MethodPost, "/api/v1/namespaces", "*", "CREATE", false},
		{"own namespace readable", http.MethodGet, "/api/v1/namespaces/default", "default", "GET", true},
		{"other namespace denied", http.MethodGet, "/api/v1/namespaces/production", "production", "GET", false},
		{"update requires permission", http.MethodPut, "/api/v1/namespaces/default", "default", "UPDATE", false},
		{"delete requires permission", http.MethodDelete, "/api/v1/namespaces/default", "default", "DELETE", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			email := "user@example.com"
			called := false
			router := gin.New()
			router.Use(func(ctx *gin.Context) {
				security.SetUser(ctx, &security.User{Authenticated: true, Email: &email})
				ctx.Next()
			})
			router.Use(security.NewAuthorizationMiddleware(namespaceRBAC{check: func(scope, resource, action string) bool {
				called = true

				assert.Equal(t, tt.scope, scope)
				assert.Equal(t, "namespace", resource)
				assert.Equal(t, tt.action, action)

				return tt.allowed
			}}, namespaceUsers{}, adminEmail, slog.Default()))

			route := "/api/v1/namespaces"
			if tt.scope != "*" {
				route += "/:namespace"
			}

			router.Handle(tt.method, route, func(ctx *gin.Context) { ctx.Status(http.StatusOK) })

			recorder := httptest.NewRecorder()
			req, err := http.NewRequestWithContext(t.Context(), tt.method, tt.path, nil)
			require.NoError(t, err)
			router.ServeHTTP(recorder, req)

			assert.True(t, called)

			if tt.allowed {
				assert.Equal(t, http.StatusOK, recorder.Code)
			} else {
				assert.Equal(t, http.StatusForbidden, recorder.Code)
			}
		})
	}
}
