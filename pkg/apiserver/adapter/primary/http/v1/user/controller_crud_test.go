package user_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/user"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/user/usecasemock"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

var errUserBoom = errors.New("boom")

func newUser(uid string) *v1.User {
	//exhaustruct:ignore
	return &v1.User{
		Kind:       v1.UserKind,
		APIVersion: v1.APIVersion,
		//exhaustruct:ignore
		Metadata: v1.UserMetadata{UID: uid},
	}
}

func crudSetup(t *testing.T) (*testutil.ControllerBase, *usecasemock.MockUsecase) {
	t.Helper()

	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := user.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)

	return ctrlBase, usecase
}

func crudReq(t *testing.T, router *gin.Engine, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), method, target, strings.NewReader(body))
	require.NoError(t, err)

	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	router.ServeHTTP(recorder, req)

	return recorder
}

func TestUserController_RoutesInfo(t *testing.T) {
	t.Parallel()

	controller := user.NewController(usecasemock.NewMockUsecase(t), testutil.NewBase(t).Logger)

	routes := controller.RoutesInfo()
	require.Len(t, routes, 5)

	got := make(map[string]struct{}, len(routes))
	for _, route := range routes {
		got[route.Method+" "+route.Path] = struct{}{}

		assert.NotNil(t, route.HandlerFunc)
	}

	for _, want := range []string{
		"GET /api/v1/users/me",
		"GET /api/v1/users",
		"GET /api/v1/users/:id",
		"POST /api/v1/users",
		"DELETE /api/v1/users/:id",
	} {
		assert.Contains(t, got, want)
	}
}

func TestUserController_List(t *testing.T) {
	t.Parallel()

	t.Run("returns the list of users", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := crudSetup(t)
		//exhaustruct:ignore
		usecase.On("ListUsers", mock.Anything, mock.Anything).
			Return(&v1.ListResponse[v1.User]{Items: []v1.User{*newUser("u-1")}}, nil)

		recorder := crudReq(t, ctrlBase.Router, http.MethodGet, "/api/v1/users?limit=10&includeDeleted=true", "")

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, int64(1), gjson.Get(recorder.Body.String(), "items.#").Int())
	})

	t.Run("returns 400 on an invalid limit", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := crudSetup(t)

		recorder := crudReq(t, ctrlBase.Router, http.MethodGet, "/api/v1/users?limit=abc", "")

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("returns 400 on an invalid includeDeleted", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := crudSetup(t)

		recorder := crudReq(t, ctrlBase.Router, http.MethodGet, "/api/v1/users?includeDeleted=maybe", "")

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("returns 500 when the usecase fails", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := crudSetup(t)
		usecase.On("ListUsers", mock.Anything, mock.Anything).Return(nil, errUserBoom)

		recorder := crudReq(t, ctrlBase.Router, http.MethodGet, "/api/v1/users", "")

		require.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}

func TestUserController_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns the user", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := crudSetup(t)
		uid := uuid.New()
		usecase.On("GetUser", mock.Anything, uid, mock.Anything).Return(newUser(uid.String()), nil)

		recorder := crudReq(t, ctrlBase.Router, http.MethodGet, "/api/v1/users/"+uid.String(), "")

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, uid.String(), gjson.Get(recorder.Body.String(), "metadata.uid").String())
	})

	t.Run("returns 400 on an invalid id", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := crudSetup(t)

		recorder := crudReq(t, ctrlBase.Router, http.MethodGet, "/api/v1/users/not-a-uuid", "")

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("returns 400 on an invalid includeDeleted", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := crudSetup(t)
		uid := uuid.New()

		recorder := crudReq(t, ctrlBase.Router, http.MethodGet, "/api/v1/users/"+uid.String()+"?includeDeleted=nope", "")

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("returns 404 when the user does not exist", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := crudSetup(t)
		uid := uuid.New()
		usecase.On("GetUser", mock.Anything, uid, mock.Anything).Return(nil, model.ErrResourceNotExist)

		recorder := crudReq(t, ctrlBase.Router, http.MethodGet, "/api/v1/users/"+uid.String(), "")

		require.Equal(t, http.StatusNotFound, recorder.Code)
	})
}

func TestUserController_Create(t *testing.T) {
	t.Parallel()

	t.Run("creates a user", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := crudSetup(t)
		usecase.On("CreateUser", mock.Anything, mock.Anything).Return(newUser("u-1"), nil)

		recorder := crudReq(t, ctrlBase.Router, http.MethodPost, "/api/v1/users", `{"spec":{"email":"a@b.c"}}`)

		require.Equal(t, http.StatusCreated, recorder.Code)
	})

	t.Run("returns 400 on an invalid body", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := crudSetup(t)

		recorder := crudReq(t, ctrlBase.Router, http.MethodPost, "/api/v1/users", `{not-json`)

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("returns 500 when the usecase fails", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := crudSetup(t)
		usecase.On("CreateUser", mock.Anything, mock.Anything).Return(nil, errUserBoom)

		recorder := crudReq(t, ctrlBase.Router, http.MethodPost, "/api/v1/users", `{"spec":{"email":"a@b.c"}}`)

		require.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}

func TestUserController_Delete(t *testing.T) {
	t.Parallel()

	t.Run("deletes a user", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := crudSetup(t)
		uid := uuid.New()
		usecase.On("DeleteUser", mock.Anything, uid).Return(nil)

		recorder := crudReq(t, ctrlBase.Router, http.MethodDelete, "/api/v1/users/"+uid.String(), "")

		require.Equal(t, http.StatusNoContent, recorder.Code)
	})

	t.Run("returns 400 on an invalid id", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := crudSetup(t)

		recorder := crudReq(t, ctrlBase.Router, http.MethodDelete, "/api/v1/users/not-a-uuid", "")

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("returns 404 when the user does not exist", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := crudSetup(t)
		uid := uuid.New()
		usecase.On("DeleteUser", mock.Anything, uid).Return(model.ErrResourceNotExist)

		recorder := crudReq(t, ctrlBase.Router, http.MethodDelete, "/api/v1/users/"+uid.String(), "")

		require.Equal(t, http.StatusNotFound, recorder.Code)
	})
}

func TestUserController_Me_InternalError(t *testing.T) {
	t.Parallel()

	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := user.NewController(usecase, ctrlBase.Logger)

	email := "alice@example.com"
	usecase.On("GetMyProfile", mock.Anything, email).Return(nil, errUserBoom)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/users/me", nil)
	require.NoError(t, err)
	routerWithAuth(controller, email).ServeHTTP(recorder, req)

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
}
