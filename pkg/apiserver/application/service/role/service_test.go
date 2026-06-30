package role_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	applicationport "github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	rolesvc "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/role"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	usermodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

var errMock = errors.New("mock error")

// mockRoleUsecase is a mock implementation of userport.RoleUsecase.
type mockRoleUsecase struct {
	mock.Mock
}

func (m *mockRoleUsecase) GetRole(
	ctx context.Context, uid uuid.UUID, options *model.GetOptions,
) (*usermodel.Role, error) {
	args := m.Called(ctx, uid, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	role, ok := args.Get(0).(*usermodel.Role)
	if !ok {
		return nil, errMock
	}

	return role, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockRoleUsecase) GetRoleByName(ctx context.Context, displayName string) (*usermodel.Role, error) {
	args := m.Called(ctx, displayName)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	role, ok := args.Get(0).(*usermodel.Role)
	if !ok {
		return nil, errMock
	}

	return role, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockRoleUsecase) ListRoles(
	ctx context.Context, options *model.ListOptions,
) (*model.ListResponse[*usermodel.Role], error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, ok := args.Get(0).(*model.ListResponse[*usermodel.Role])
	if !ok {
		return nil, errMock
	}

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockRoleUsecase) SaveRole(ctx context.Context, role *usermodel.Role) error {
	args := m.Called(ctx, role)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func (m *mockRoleUsecase) CreateRole(ctx context.Context, role *usermodel.Role) (*usermodel.Role, error) {
	args := m.Called(ctx, role)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	r, ok := args.Get(0).(*usermodel.Role)
	if !ok {
		return nil, errMock
	}

	return r, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockRoleUsecase) UpdateRole(
	ctx context.Context, uid uuid.UUID, role *usermodel.Role,
) (*usermodel.Role, error) {
	args := m.Called(ctx, uid, role)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	r, ok := args.Get(0).(*usermodel.Role)
	if !ok {
		return nil, errMock
	}

	return r, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockRoleUsecase) DeleteRole(ctx context.Context, uid uuid.UUID) error {
	args := m.Called(ctx, uid)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func newSvc(t *testing.T, role *mockRoleUsecase) *rolesvc.Service {
	t.Helper()

	base := testutil.NewBase(t)

	return rolesvc.New(role, base.Logger)
}

func newRole() *usermodel.Role {
	role := usermodel.NewRole("Viewer", false)
	role.Spec.Permissions = []string{"agent:read"}

	return role
}

func apiRole() *v1.Role {
	return &v1.Role{
		Kind:       v1.RoleKind,
		APIVersion: v1.APIVersion,
		Spec:       v1.RoleSpec{DisplayName: "Viewer", Permissions: []string{"agent:read"}},
	}
}

func TestService_GetRole(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockRole := new(mockRoleUsecase)
		svc := newSvc(t, mockRole)

		uid := uuid.New()
		mockRole.On("GetRole", ctx, uid, (*model.GetOptions)(nil)).Return(newRole(), nil)

		result, err := svc.GetRole(ctx, uid, nil)

		require.NoError(t, err)
		assert.Equal(t, v1.RoleKind, result.Kind)
		assert.Equal(t, "Viewer", result.Spec.DisplayName)
		mockRole.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockRole := new(mockRoleUsecase)
		svc := newSvc(t, mockRole)

		uid := uuid.New()
		mockRole.On("GetRole", ctx, uid, (*model.GetOptions)(nil)).Return(nil, errMock)

		result, err := svc.GetRole(ctx, uid, nil)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to get role")
		mockRole.AssertExpectations(t)
	})
}

func TestService_ListRoles(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockRole := new(mockRoleUsecase)
		svc := newSvc(t, mockRole)

		opts := &applicationport.ListOptions{Limit: 10}
		resp := &model.ListResponse[*usermodel.Role]{
			Items:    []*usermodel.Role{newRole()},
			Continue: "next",
		}
		mockRole.On("ListRoles", ctx, opts.ToDomain()).Return(resp, nil)

		result, err := svc.ListRoles(ctx, opts)

		require.NoError(t, err)
		assert.Equal(t, v1.RoleKind, result.Kind)
		assert.Len(t, result.Items, 1)
		assert.Equal(t, "next", result.Metadata.Continue)
		mockRole.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockRole := new(mockRoleUsecase)
		svc := newSvc(t, mockRole)

		opts := &applicationport.ListOptions{Limit: 10}
		mockRole.On("ListRoles", ctx, opts.ToDomain()).Return(nil, errMock)

		result, err := svc.ListRoles(ctx, opts)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to list roles")
		mockRole.AssertExpectations(t)
	})
}

func TestService_CreateRole(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockRole := new(mockRoleUsecase)
		svc := newSvc(t, mockRole)

		mockRole.On("CreateRole", ctx, mock.MatchedBy(func(r *usermodel.Role) bool {
			return r.Spec.DisplayName == "Viewer" && len(r.Spec.Permissions) == 1
		})).Return(newRole(), nil)

		result, err := svc.CreateRole(ctx, apiRole())

		require.NoError(t, err)
		assert.Equal(t, "Viewer", result.Spec.DisplayName)
		mockRole.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockRole := new(mockRoleUsecase)
		svc := newSvc(t, mockRole)

		mockRole.On("CreateRole", ctx, mock.Anything).Return(nil, errMock)

		result, err := svc.CreateRole(ctx, apiRole())

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to create role")
		mockRole.AssertExpectations(t)
	})
}

func TestService_UpdateRole(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockRole := new(mockRoleUsecase)
		svc := newSvc(t, mockRole)

		uid := uuid.New()
		mockRole.On("UpdateRole", ctx, uid, mock.Anything).Return(newRole(), nil)

		result, err := svc.UpdateRole(ctx, uid, apiRole())

		require.NoError(t, err)
		assert.Equal(t, "Viewer", result.Spec.DisplayName)
		mockRole.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockRole := new(mockRoleUsecase)
		svc := newSvc(t, mockRole)

		uid := uuid.New()
		mockRole.On("UpdateRole", ctx, uid, mock.Anything).Return(nil, errMock)

		result, err := svc.UpdateRole(ctx, uid, apiRole())

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to update role")
		mockRole.AssertExpectations(t)
	})
}

func TestService_DeleteRole(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockRole := new(mockRoleUsecase)
		svc := newSvc(t, mockRole)

		uid := uuid.New()
		mockRole.On("DeleteRole", ctx, uid).Return(nil)

		err := svc.DeleteRole(ctx, uid)

		require.NoError(t, err)
		mockRole.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockRole := new(mockRoleUsecase)
		svc := newSvc(t, mockRole)

		uid := uuid.New()
		mockRole.On("DeleteRole", ctx, uid).Return(errMock)

		err := svc.DeleteRole(ctx, uid)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete role")
		mockRole.AssertExpectations(t)
	})
}
