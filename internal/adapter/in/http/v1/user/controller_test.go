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

func TestUserController_GetMyRoles(t *testing.T) {
	t.Parallel()

	t.Run("returns roles for authenticated user", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		userUsecase := usecasemock.NewMockUsecase(t)
		rbacUsecase := usecasemock.NewMockRBACUsecase(t)
		controller := user.NewController(userUsecase, rbacUsecase, ctrlBase.Logger)

		email := "alice@example.com"

		rbacUsecase.On("GetMyRoles", mock.Anything, email).Return(&v1.ListResponse[v1.Role]{
			Kind:       v1.RoleKind,
			APIVersion: v1.APIVersion,
			Items: []v1.Role{
				{Spec: v1.RoleSpec{DisplayName: "default"}},
			},
		}, nil)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/users/me/roles", nil)
		require.NoError(t, err)
		routerWithAuth(controller, email).ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, int64(1), gjson.Get(recorder.Body.String(), "items.#").Int())
		assert.Equal(t, "default", gjson.Get(recorder.Body.String(), "items.0.spec.displayName").String())
	})

	t.Run("returns 401 when unauthenticated", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		userUsecase := usecasemock.NewMockUsecase(t)
		rbacUsecase := usecasemock.NewMockRBACUsecase(t)
		controller := user.NewController(userUsecase, rbacUsecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/users/me/roles", nil)
		require.NoError(t, err)
		ctrlBase.Router.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	})
}

func TestUserController_GetMyRoleBindings(t *testing.T) {
	t.Parallel()

	t.Run("returns matching role bindings for authenticated user", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		userUsecase := usecasemock.NewMockUsecase(t)
		rbacUsecase := usecasemock.NewMockRBACUsecase(t)
		controller := user.NewController(userUsecase, rbacUsecase, ctrlBase.Logger)

		email := "bob@example.com"

		rbacUsecase.On("GetMyRoleBindings", mock.Anything, email).Return(&v1.ListResponse[v1.RoleBinding]{
			Kind:       v1.RoleBindingKind,
			APIVersion: v1.APIVersion,
			Items: []v1.RoleBinding{
				{
					Metadata: v1.RoleBindingMetadata{Namespace: "default", Name: "viewer-binding"},
					Spec: v1.RoleBindingSpec{
						RoleRef:       v1.RoleBindingRoleRef{Kind: "Role", Name: "Viewer"},
						LabelSelector: map[string]string{"login-type": "github"},
					},
				},
			},
		}, nil)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/users/me/rolebindings", nil)
		require.NoError(t, err)
		routerWithAuth(controller, email).ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, int64(1), gjson.Get(recorder.Body.String(), "items.#").Int())
		assert.Equal(t, "viewer-binding", gjson.Get(recorder.Body.String(), "items.0.metadata.name").String())
	})

	t.Run("returns empty list when no matching bindings", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		userUsecase := usecasemock.NewMockUsecase(t)
		rbacUsecase := usecasemock.NewMockRBACUsecase(t)
		controller := user.NewController(userUsecase, rbacUsecase, ctrlBase.Logger)

		email := "nobody@example.com"

		rbacUsecase.On("GetMyRoleBindings", mock.Anything, email).Return(&v1.ListResponse[v1.RoleBinding]{
			Kind:       v1.RoleBindingKind,
			APIVersion: v1.APIVersion,
			Items:      []v1.RoleBinding{},
		}, nil)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/users/me/rolebindings", nil)
		require.NoError(t, err)
		routerWithAuth(controller, email).ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, int64(0), gjson.Get(recorder.Body.String(), "items.#").Int())
	})
}
