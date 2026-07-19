package basic_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"go.uber.org/goleak"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/auth/basic"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/inmemory"
	applicationport "github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/usecase"
	usermodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user"
	userport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/security"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	// memguard starts a single process-lifetime daemon (the Coffer rekeying goroutine) the
	// first time an enclave is created; it never exits by design, so it is not a leak.
	goleak.VerifyTestMain(m,
		goleak.IgnoreTopFunction("github.com/awnumar/memguard/core.NewCoffer.func1"),
	)
}

const (
	adminUsername = "admin"
	adminPassword = "adminpass"
	adminEmail    = "admin@example.com"
	testPepper    = "test-pepper"
)

// errBoom is a non-sentinel error used to drive the service into its 500 path.
var errBoom = errors.New("boom")

// provisioningSpy is a fake AuthProvisioningUsecase that records the calls it receives.
type provisioningSpy struct {
	calls []applicationport.LoginProvisioning
}

var _ usecase.AuthProvisioningUsecase = (*provisioningSpy)(nil)

func (s *provisioningSpy) EnsureUserOnLogin(_ context.Context, provisioning applicationport.LoginProvisioning) {
	s.calls = append(s.calls, provisioning)
}

// erroringUserRepo returns a generic error from GetUserByUsername so the security
// service surfaces a non-credential error (→ HTTP 500).
type erroringUserRepo struct {
	*inmemory.UserRepository
}

var _ userport.UserPersistencePort = (*erroringUserRepo)(nil)

func (r *erroringUserRepo) GetUserByUsername(_ context.Context, _ string) (*usermodel.User, error) {
	return nil, errBoom
}

// newService builds a real security.Service backed by the provided user repository.
func newService(t *testing.T, repo userport.UserPersistencePort, refreshExpiration time.Duration) *security.Service {
	t.Helper()

	//exhaustruct:ignore
	cfg := &security.Config{
		BasicAuthSettings: security.BasicAuthSettings{Pepper: testPepper},
		//exhaustruct:ignore
		JWTSettings: security.JWTSettings{
			SigningKey:        "test-signing-key",
			Issuer:            "test",
			Expiration:        time.Minute,
			RefreshExpiration: refreshExpiration,
		},
		AdminSettings: security.AdminSettings{
			Username: adminUsername,
			Password: adminPassword,
			Email:    adminEmail,
		},
	}

	return security.New(slog.Default(), cfg, http.DefaultClient, security.NewPasswordHasher(cfg), repo)
}

// newRouter registers the controller routes without any authentication middleware.
func newRouter(controller *basic.Controller) *gin.Engine {
	router := gin.New()
	for _, route := range controller.RoutesInfo() {
		router.Handle(route.Method, route.Path, route.HandlerFunc)
	}

	return router
}

// newAuthedRouter registers the controller routes behind a middleware that injects
// an authenticated user into the context.
func newAuthedRouter(controller *basic.Controller, user *security.User) *gin.Engine {
	router := gin.New()

	router.Use(func(ctx *gin.Context) {
		if user != nil {
			security.SetUser(ctx, user)
		}

		ctx.Next()
	})

	for _, route := range controller.RoutesInfo() {
		router.Handle(route.Method, route.Path, route.HandlerFunc)
	}

	return router
}

func TestController_RoutesInfo(t *testing.T) {
	t.Parallel()

	controller := basic.NewController(slog.Default(), newService(t, inmemory.NewUserRepository(), 0), &provisioningSpy{})
	require.NotNil(t, controller)

	routes := controller.RoutesInfo()
	require.Len(t, routes, 3)

	got := make(map[string]string, len(routes))
	for _, route := range routes {
		got[route.Method+" "+route.Path] = route.Handler
		assert.NotNil(t, route.HandlerFunc)
	}

	assert.Contains(t, got, "GET /api/v1/auth/basic")
	assert.Contains(t, got, "GET /api/v1/auth/info")
	assert.Contains(t, got, "POST /api/v1/auth/refresh")
}

func TestController_BasicAuth(t *testing.T) {
	t.Parallel()

	t.Run("returns token and provisions user on valid admin credentials", func(t *testing.T) {
		t.Parallel()

		spy := &provisioningSpy{}
		controller := basic.NewController(slog.Default(), newService(t, inmemory.NewUserRepository(), 0), spy)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/auth/basic", nil)
		require.NoError(t, err)
		req.SetBasicAuth(adminUsername, adminPassword)

		newRouter(controller).ServeHTTP(recorder, req)

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.NotEmpty(t, gjson.Get(recorder.Body.String(), "token").String())

		require.Len(t, spy.calls, 1)
		assert.Equal(t, applicationport.IdentityProviderBasic, spy.calls[0].Provider)
		assert.Equal(t, adminUsername, spy.calls[0].Username)
		assert.Equal(t, adminEmail, spy.calls[0].Email)
		assert.Nil(t, spy.calls[0].Groups)
	})

	t.Run("returns 401 when basic auth header is missing", func(t *testing.T) {
		t.Parallel()

		spy := &provisioningSpy{}
		controller := basic.NewController(slog.Default(), newService(t, inmemory.NewUserRepository(), 0), spy)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/auth/basic", nil)
		require.NoError(t, err)

		newRouter(controller).ServeHTTP(recorder, req)

		require.Equal(t, http.StatusUnauthorized, recorder.Code)
		assert.Equal(t, "missing basic auth credentials", gjson.Get(recorder.Body.String(), "error").String())
		assert.Empty(t, spy.calls)
	})

	t.Run("returns 401 on invalid credentials", func(t *testing.T) {
		t.Parallel()

		spy := &provisioningSpy{}
		controller := basic.NewController(slog.Default(), newService(t, inmemory.NewUserRepository(), 0), spy)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/auth/basic", nil)
		require.NoError(t, err)
		req.SetBasicAuth(adminUsername, "wrong-password")

		newRouter(controller).ServeHTTP(recorder, req)

		require.Equal(t, http.StatusUnauthorized, recorder.Code)
		assert.Equal(t, "invalid username or password", gjson.Get(recorder.Body.String(), "error").String())
		assert.Empty(t, spy.calls)
	})

	t.Run("returns 500 when the service fails unexpectedly", func(t *testing.T) {
		t.Parallel()

		repo := &erroringUserRepo{UserRepository: inmemory.NewUserRepository()}
		spy := &provisioningSpy{}
		controller := basic.NewController(slog.Default(), newService(t, repo, 0), spy)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/auth/basic", nil)
		require.NoError(t, err)
		req.SetBasicAuth("someone", "whatever")

		newRouter(controller).ServeHTTP(recorder, req)

		require.Equal(t, http.StatusInternalServerError, recorder.Code)
		assert.Equal(t, "failed to authenticate", gjson.Get(recorder.Body.String(), "error").String())
		assert.Empty(t, spy.calls)
	})
}

func TestController_Refresh(t *testing.T) {
	t.Parallel()

	t.Run("exchanges a valid refresh token for a new token", func(t *testing.T) {
		t.Parallel()

		service := newService(t, inmemory.NewUserRepository(), time.Hour)
		controller := basic.NewController(slog.Default(), service, &provisioningSpy{})

		// Mint a refresh token through the real service.
		login, err := service.BasicAuth(t.Context(), adminUsername, adminPassword)
		require.NoError(t, err)
		require.NotEmpty(t, login.RefreshToken)

		recorder := httptest.NewRecorder()
		body := strings.NewReader(`{"refreshToken":"` + login.RefreshToken + `"}`)
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/auth/refresh", body)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		newRouter(controller).ServeHTTP(recorder, req)

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.NotEmpty(t, gjson.Get(recorder.Body.String(), "token").String())
		assert.NotEmpty(t, gjson.Get(recorder.Body.String(), "refreshToken").String())
	})

	t.Run("returns 400 on malformed request body", func(t *testing.T) {
		t.Parallel()

		controller := basic.NewController(slog.Default(), newService(t, inmemory.NewUserRepository(), 0), &provisioningSpy{})

		recorder := httptest.NewRecorder()
		body := strings.NewReader(`{not-json`)
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/auth/refresh", body)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		newRouter(controller).ServeHTTP(recorder, req)

		require.Equal(t, http.StatusBadRequest, recorder.Code)
		assert.Equal(t, "invalid request body", gjson.Get(recorder.Body.String(), "error").String())
	})

	t.Run("returns 400 when refresh token is missing", func(t *testing.T) {
		t.Parallel()

		controller := basic.NewController(slog.Default(), newService(t, inmemory.NewUserRepository(), 0), &provisioningSpy{})

		recorder := httptest.NewRecorder()
		body := strings.NewReader(`{}`)
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/auth/refresh", body)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		newRouter(controller).ServeHTTP(recorder, req)

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("returns 401 on an invalid refresh token", func(t *testing.T) {
		t.Parallel()

		controller := basic.NewController(slog.Default(), newService(t, inmemory.NewUserRepository(), 0), &provisioningSpy{})

		recorder := httptest.NewRecorder()
		body := strings.NewReader(`{"refreshToken":"not-a-valid-token"}`)
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/auth/refresh", body)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		newRouter(controller).ServeHTTP(recorder, req)

		require.Equal(t, http.StatusUnauthorized, recorder.Code)
		assert.Equal(t, "invalid or expired refresh token", gjson.Get(recorder.Body.String(), "error").String())
	})
}

func TestController_Info(t *testing.T) {
	t.Parallel()

	t.Run("returns auth info for an authenticated user", func(t *testing.T) {
		t.Parallel()

		controller := basic.NewController(slog.Default(), newService(t, inmemory.NewUserRepository(), 0), &provisioningSpy{})

		email := "alice@example.com"
		user := &security.User{Authenticated: true, Email: &email}

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/auth/info", nil)
		require.NoError(t, err)

		newAuthedRouter(controller, user).ServeHTTP(recorder, req)

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.True(t, gjson.Get(recorder.Body.String(), "authenticated").Bool())
		assert.Equal(t, email, gjson.Get(recorder.Body.String(), "email").String())
	})

	t.Run("returns 401 when no user is in the context", func(t *testing.T) {
		t.Parallel()

		controller := basic.NewController(slog.Default(), newService(t, inmemory.NewUserRepository(), 0), &provisioningSpy{})

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/auth/info", nil)
		require.NoError(t, err)

		newAuthedRouter(controller, nil).ServeHTTP(recorder, req)

		require.Equal(t, http.StatusUnauthorized, recorder.Code)
		assert.Equal(t, "unauthorized", gjson.Get(recorder.Body.String(), "error").String())
	})
}
