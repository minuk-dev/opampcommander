package role_test

import (
	"context"
	"errors"
	"log/slog"
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
	"go.uber.org/goleak"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/role"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	goleak.VerifyTestMain(m)
}

var errBoom = errors.New("boom")

// mockRoleUsecase is a testify mock of role.Usecase (usecase.RoleManageUsecase).
type mockRoleUsecase struct {
	mock.Mock
}

func newMockRoleUsecase(t *testing.T) *mockRoleUsecase {
	t.Helper()

	m := &mockRoleUsecase{}
	m.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}

func (m *mockRoleUsecase) GetRole(ctx context.Context, uid uuid.UUID, options *port.GetOptions) (*v1.Role, error) {
	args := m.Called(ctx, uid, options)

	res, _ := args.Get(0).(*v1.Role)

	return res, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockRoleUsecase) ListRoles(ctx context.Context, options *port.ListOptions) (*v1.ListResponse[v1.Role], error) {
	args := m.Called(ctx, options)

	res, _ := args.Get(0).(*v1.ListResponse[v1.Role])

	return res, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockRoleUsecase) CreateRole(ctx context.Context, r *v1.Role) (*v1.Role, error) {
	args := m.Called(ctx, r)

	res, _ := args.Get(0).(*v1.Role)

	return res, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockRoleUsecase) UpdateRole(ctx context.Context, uid uuid.UUID, r *v1.Role) (*v1.Role, error) {
	args := m.Called(ctx, uid, r)

	res, _ := args.Get(0).(*v1.Role)

	return res, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockRoleUsecase) DeleteRole(ctx context.Context, uid uuid.UUID) error {
	args := m.Called(ctx, uid)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func newRole(uid string) *v1.Role {
	//exhaustruct:ignore
	return &v1.Role{
		Kind:       v1.RoleKind,
		APIVersion: v1.APIVersion,
		//exhaustruct:ignore
		Metadata: v1.RoleMetadata{UID: uid},
	}
}

func setup(t *testing.T) (*testutil.ControllerBase, *mockRoleUsecase) {
	t.Helper()

	ctrlBase := testutil.NewBase(t).ForController()
	usecase := newMockRoleUsecase(t)
	controller := role.NewController(usecase, slog.Default())
	ctrlBase.SetupRouter(controller)

	return ctrlBase, usecase
}

func doReq(t *testing.T, router *gin.Engine, method, target, body string) *httptest.ResponseRecorder {
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

func TestRoleController_RoutesInfo(t *testing.T) {
	t.Parallel()

	controller := role.NewController(newMockRoleUsecase(t), slog.Default())

	routes := controller.RoutesInfo()
	require.Len(t, routes, 5)

	got := make(map[string]struct{}, len(routes))
	for _, route := range routes {
		got[route.Method+" "+route.Path] = struct{}{}

		assert.NotNil(t, route.HandlerFunc)
	}

	for _, want := range []string{
		"GET /api/v1/roles",
		"GET /api/v1/roles/:id",
		"POST /api/v1/roles",
		"PUT /api/v1/roles/:id",
		"DELETE /api/v1/roles/:id",
	} {
		assert.Contains(t, got, want)
	}
}

func TestRoleController_List(t *testing.T) {
	t.Parallel()

	t.Run("returns the list of roles", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		//exhaustruct:ignore
		usecase.On("ListRoles", mock.Anything, mock.Anything).Return(&v1.ListResponse[v1.Role]{
			Items: []v1.Role{*newRole(uuid.NewString())},
		}, nil)

		recorder := doReq(t, ctrlBase.Router, http.MethodGet, "/api/v1/roles?limit=10&includeDeleted=true", "")

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, int64(1), gjson.Get(recorder.Body.String(), "items.#").Int())
	})

	t.Run("returns 400 on an invalid limit", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := setup(t)

		recorder := doReq(t, ctrlBase.Router, http.MethodGet, "/api/v1/roles?limit=abc", "")

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("returns 400 on an invalid includeDeleted", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := setup(t)

		recorder := doReq(t, ctrlBase.Router, http.MethodGet, "/api/v1/roles?includeDeleted=maybe", "")

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("returns 500 when the usecase fails", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("ListRoles", mock.Anything, mock.Anything).Return(nil, errBoom)

		recorder := doReq(t, ctrlBase.Router, http.MethodGet, "/api/v1/roles", "")

		require.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}

func TestRoleController_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns the role", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		uid := uuid.New()
		usecase.On("GetRole", mock.Anything, uid, mock.Anything).Return(newRole(uid.String()), nil)

		recorder := doReq(t, ctrlBase.Router, http.MethodGet, "/api/v1/roles/"+uid.String(), "")

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, uid.String(), gjson.Get(recorder.Body.String(), "metadata.uid").String())
	})

	t.Run("returns 400 on an invalid id", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := setup(t)

		recorder := doReq(t, ctrlBase.Router, http.MethodGet, "/api/v1/roles/not-a-uuid", "")

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("returns 400 on an invalid includeDeleted", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := setup(t)

		recorder := doReq(t, ctrlBase.Router, http.MethodGet,
			"/api/v1/roles/"+uuid.NewString()+"?includeDeleted=nope", "")

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("returns 404 when the role does not exist", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		uid := uuid.New()
		usecase.On("GetRole", mock.Anything, uid, mock.Anything).Return(nil, model.ErrResourceNotExist)

		recorder := doReq(t, ctrlBase.Router, http.MethodGet, "/api/v1/roles/"+uid.String(), "")

		require.Equal(t, http.StatusNotFound, recorder.Code)
	})
}

func TestRoleController_Create(t *testing.T) {
	t.Parallel()

	t.Run("creates a role", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("CreateRole", mock.Anything, mock.Anything).Return(newRole(uuid.NewString()), nil)

		recorder := doReq(t, ctrlBase.Router, http.MethodPost, "/api/v1/roles", `{"metadata":{}}`)

		require.Equal(t, http.StatusCreated, recorder.Code)
	})

	t.Run("returns 400 on an invalid body", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := setup(t)

		recorder := doReq(t, ctrlBase.Router, http.MethodPost, "/api/v1/roles", `{not-json`)

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("returns 500 when the usecase fails", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		usecase.On("CreateRole", mock.Anything, mock.Anything).Return(nil, errBoom)

		recorder := doReq(t, ctrlBase.Router, http.MethodPost, "/api/v1/roles", `{"metadata":{}}`)

		require.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}

func TestRoleController_Update(t *testing.T) {
	t.Parallel()

	t.Run("updates a role", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		uid := uuid.New()
		usecase.On("UpdateRole", mock.Anything, uid, mock.Anything).Return(newRole(uid.String()), nil)

		recorder := doReq(t, ctrlBase.Router, http.MethodPut, "/api/v1/roles/"+uid.String(), `{"metadata":{}}`)

		require.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("returns 400 on an invalid id", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := setup(t)

		recorder := doReq(t, ctrlBase.Router, http.MethodPut, "/api/v1/roles/not-a-uuid", `{"metadata":{}}`)

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("returns 400 on an invalid body", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := setup(t)

		recorder := doReq(t, ctrlBase.Router, http.MethodPut, "/api/v1/roles/"+uuid.NewString(), `{not-json`)

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("returns 404 when the role does not exist", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		uid := uuid.New()
		usecase.On("UpdateRole", mock.Anything, uid, mock.Anything).Return(nil, model.ErrResourceNotExist)

		recorder := doReq(t, ctrlBase.Router, http.MethodPut, "/api/v1/roles/"+uid.String(), `{"metadata":{}}`)

		require.Equal(t, http.StatusNotFound, recorder.Code)
	})
}

func TestRoleController_Delete(t *testing.T) {
	t.Parallel()

	t.Run("deletes a role", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		uid := uuid.New()
		usecase.On("DeleteRole", mock.Anything, uid).Return(nil)

		recorder := doReq(t, ctrlBase.Router, http.MethodDelete, "/api/v1/roles/"+uid.String(), "")

		require.Equal(t, http.StatusNoContent, recorder.Code)
	})

	t.Run("returns 400 on an invalid id", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := setup(t)

		recorder := doReq(t, ctrlBase.Router, http.MethodDelete, "/api/v1/roles/not-a-uuid", "")

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("returns 404 when the role does not exist", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := setup(t)
		uid := uuid.New()
		usecase.On("DeleteRole", mock.Anything, uid).Return(model.ErrResourceNotExist)

		recorder := doReq(t, ctrlBase.Router, http.MethodDelete, "/api/v1/roles/"+uid.String(), "")

		require.Equal(t, http.StatusNotFound, recorder.Code)
	})
}
