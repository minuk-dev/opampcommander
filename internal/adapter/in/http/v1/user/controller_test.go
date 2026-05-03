package user_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"go.uber.org/goleak"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/user"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/user/usecasemock"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/internal/security"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestMain(m *testing.M) { goleak.VerifyTestMain(m) }

// routerWithAuth builds a gin engine that injects an authenticated user before routing.
func routerWithAuth(controller *user.Controller, email string) *gin.Engine {
	router := gin.New()

	router.Use(func(ctx *gin.Context) {
		security.SetUser(ctx, &security.User{
			Authenticated: true,
			Email:         &email,
		})
		ctx.Next()
	})

	for _, route := range controller.RoutesInfo() {
		router.Handle(route.Method, route.Path, route.HandlerFunc)
	}

	return router
}

func TestUserController_Me(t *testing.T) {
	t.Parallel()

	t.Run("returns profile with roles for authenticated user", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		userUsecase := usecasemock.NewMockUsecase(t)
		controller := user.NewController(userUsecase, ctrlBase.Logger)

		email := "alice@example.com"

		userUsecase.On("GetMyProfile", mock.Anything, email).Return(&v1.UserProfileResponse{
			User: v1.User{
				Kind:       v1.UserKind,
				APIVersion: "v1",
				Spec: v1.UserSpec{
					Email:    email,
					Username: "alice",
					IsActive: true,
				},
			},
			Roles: []v1.UserRoleEntry{
				{
					Role: v1.Role{Spec: v1.RoleSpec{DisplayName: "default", IsBuiltIn: true}},
				},
			},
		}, nil)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/users/me", nil)
		require.NoError(t, err)
		routerWithAuth(controller, email).ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, email, gjson.Get(recorder.Body.String(), "user.spec.email").String())
		assert.Equal(t, "alice", gjson.Get(recorder.Body.String(), "user.spec.username").String())
		assert.Equal(t, int64(1), gjson.Get(recorder.Body.String(), "roles.#").Int())
		assert.Equal(t, "default", gjson.Get(recorder.Body.String(), "roles.0.role.spec.displayName").String())
	})

	t.Run("returns 401 when unauthenticated", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		userUsecase := usecasemock.NewMockUsecase(t)
		controller := user.NewController(userUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/users/me", nil)
		require.NoError(t, err)
		ctrlBase.Router.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	})

	t.Run("returns synthetic profile when user not in database", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		userUsecase := usecasemock.NewMockUsecase(t)
		controller := user.NewController(userUsecase, ctrlBase.Logger)

		email := "admin@example.com"

		userUsecase.On("GetMyProfile", mock.Anything, email).Return(nil, port.ErrResourceNotExist)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/users/me", nil)
		require.NoError(t, err)
		routerWithAuth(controller, email).ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, email, gjson.Get(recorder.Body.String(), "user.spec.email").String())
		assert.True(t, gjson.Get(recorder.Body.String(), "user.spec.isActive").Bool())
		assert.Empty(t, gjson.Get(recorder.Body.String(), "user.metadata.uid").String())
	})
}
