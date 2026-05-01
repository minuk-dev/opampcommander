package rolebinding_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	rolebindingsvc "github.com/minuk-dev/opampcommander/internal/application/service/rolebinding"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	usermodel "github.com/minuk-dev/opampcommander/internal/domain/user/model"
	userport "github.com/minuk-dev/opampcommander/internal/domain/user/port"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

var (
	errMock     = errors.New("mock error")
	errNotFound = errors.New("not found")
)

// mockRoleBindingUsecase is a mock implementation of RoleBindingUsecase.
type mockRoleBindingUsecase struct {
	mock.Mock
}

var _ userport.RoleBindingUsecase = (*mockRoleBindingUsecase)(nil)

func (m *mockRoleBindingUsecase) GetRoleBinding(
	ctx context.Context,
	namespace, name string,
) (*usermodel.RoleBinding, error) {
	args := m.Called(ctx, namespace, name)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	rb, ok := args.Get(0).(*usermodel.RoleBinding)
	if !ok {
		return nil, errMock
	}

	return rb, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockRoleBindingUsecase) ListRoleBindings(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*usermodel.RoleBinding], error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, ok := args.Get(0).(*model.ListResponse[*usermodel.RoleBinding])
	if !ok {
		return nil, errMock
	}

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockRoleBindingUsecase) CreateRoleBinding(
	ctx context.Context,
	rb *usermodel.RoleBinding,
) (*usermodel.RoleBinding, error) {
	args := m.Called(ctx, rb)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	result, ok := args.Get(0).(*usermodel.RoleBinding)
	if !ok {
		return nil, errMock
	}

	return result, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockRoleBindingUsecase) UpdateRoleBinding(
	ctx context.Context,
	namespace, name string,
	rb *usermodel.RoleBinding,
) (*usermodel.RoleBinding, error) {
	args := m.Called(ctx, namespace, name, rb)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	result, ok := args.Get(0).(*usermodel.RoleBinding)
	if !ok {
		return nil, errMock
	}

	return result, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockRoleBindingUsecase) DeleteRoleBinding(
	ctx context.Context,
	namespace, name string,
) error {
	args := m.Called(ctx, namespace, name)

	return args.Error(0) //nolint:wrapcheck // mock error
}

// mockRoleUsecase is a mock implementation of RoleUsecase.
type mockRoleUsecase struct {
	mock.Mock
}

var _ userport.RoleUsecase = (*mockRoleUsecase)(nil)

func (m *mockRoleUsecase) GetRole(ctx context.Context, uid uuid.UUID) (*usermodel.Role, error) {
	args := m.Called(ctx, uid)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	role, ok := args.Get(0).(*usermodel.Role)
	if !ok {
		return nil, errMock
	}

	return role, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockRoleUsecase) GetRoleByName(
	ctx context.Context,
	displayName string,
) (*usermodel.Role, error) {
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
	ctx context.Context,
	options *model.ListOptions,
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

func (m *mockRoleUsecase) DeleteRole(ctx context.Context, uid uuid.UUID) error {
	args := m.Called(ctx, uid)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func newSvc(t *testing.T, rb *mockRoleBindingUsecase, role *mockRoleUsecase) *rolebindingsvc.Service {
	t.Helper()

	base := testutil.NewBase(t)

	return rolebindingsvc.New(rb, role, base.Logger)
}

func newRB() *usermodel.RoleBinding {
	rb := usermodel.NewRoleBinding("production", "viewer-binding", usermodel.RoleRef{Kind: "Role", Name: "Viewer"})
	rb.Spec.LabelSelector = map[string]string{"login-type": "github"}

	return rb
}

func TestService_GetRoleBinding(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockRBUsecase := new(mockRoleBindingUsecase)
		mockRole := new(mockRoleUsecase)
		svc := newSvc(t, mockRBUsecase, mockRole)

		rb := newRB()
		mockRBUsecase.On("GetRoleBinding", ctx, "production", "viewer-binding").Return(rb, nil)

		result, err := svc.GetRoleBinding(ctx, "production", "viewer-binding")

		require.NoError(t, err)
		assert.Equal(t, "production", result.Metadata.Namespace)
		assert.Equal(t, "viewer-binding", result.Metadata.Name)
		mockRBUsecase.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockRBUsecase := new(mockRoleBindingUsecase)
		mockRole := new(mockRoleUsecase)
		svc := newSvc(t, mockRBUsecase, mockRole)

		mockRBUsecase.On("GetRoleBinding", ctx, "production", "viewer-binding").
			Return(nil, errMock)

		result, err := svc.GetRoleBinding(ctx, "production", "viewer-binding")

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "get role binding")
		mockRBUsecase.AssertExpectations(t)
	})
}

func TestService_ListRoleBindings(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockRBUsecase := new(mockRoleBindingUsecase)
		mockRole := new(mockRoleUsecase)
		svc := newSvc(t, mockRBUsecase, mockRole)

		rb := newRB()
		domainResp := &model.ListResponse[*usermodel.RoleBinding]{Items: []*usermodel.RoleBinding{rb}}

		opts := &model.ListOptions{Limit: 10}
		mockRBUsecase.On("ListRoleBindings", ctx, opts).Return(domainResp, nil)

		result, err := svc.ListRoleBindings(ctx, opts)

		require.NoError(t, err)
		assert.Equal(t, v1.RoleBindingKind, result.Kind)
		assert.Len(t, result.Items, 1)
		mockRBUsecase.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockRBUsecase := new(mockRoleBindingUsecase)
		mockRole := new(mockRoleUsecase)
		svc := newSvc(t, mockRBUsecase, mockRole)

		opts := &model.ListOptions{Limit: 10}
		mockRBUsecase.On("ListRoleBindings", ctx, opts).Return(nil, errMock)

		result, err := svc.ListRoleBindings(ctx, opts)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "list role bindings")
		mockRBUsecase.AssertExpectations(t)
	})
}

func TestService_CreateRoleBinding(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockRBUsecase := new(mockRoleBindingUsecase)
		mockRole := new(mockRoleUsecase)
		svc := newSvc(t, mockRBUsecase, mockRole)

		roleUID := uuid.New()
		role := &usermodel.Role{
			Metadata: usermodel.RoleMetadata{UID: roleUID},
			Spec:     usermodel.RoleSpec{DisplayName: "Viewer"},
		}

		apiRB := &v1.RoleBinding{
			Kind:       v1.RoleBindingKind,
			APIVersion: v1.APIVersion,
			Metadata:   v1.RoleBindingMetadata{Namespace: "production", Name: "viewer-binding"},
			Spec: v1.RoleBindingSpec{
				RoleRef:       v1.RoleBindingRoleRef{Kind: "Role", Name: "Viewer"},
				LabelSelector: map[string]string{"login-type": "github"},
			},
		}

		mockRole.On("GetRoleByName", ctx, "Viewer").Return(role, nil)
		mockRBUsecase.On("CreateRoleBinding", ctx, mock.MatchedBy(func(rb *usermodel.RoleBinding) bool {
			return rb.Spec.RoleRef.Name == "Viewer" && rb.Spec.LabelSelector["login-type"] == "github"
		})).Return(newRB(), nil)

		result, err := svc.CreateRoleBinding(ctx, apiRB)

		require.NoError(t, err)
		assert.Equal(t, "production", result.Metadata.Namespace)
		assert.Equal(t, "viewer-binding", result.Metadata.Name)
		mockRole.AssertExpectations(t)
		mockRBUsecase.AssertExpectations(t)
	})

	t.Run("role not found error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockRBUsecase := new(mockRoleBindingUsecase)
		mockRole := new(mockRoleUsecase)
		svc := newSvc(t, mockRBUsecase, mockRole)

		apiRB := &v1.RoleBinding{
			Spec: v1.RoleBindingSpec{
				RoleRef:       v1.RoleBindingRoleRef{Kind: "Role", Name: "NonExistent"},
				LabelSelector: map[string]string{"login-type": "github"},
			},
		}

		mockRole.On("GetRoleByName", ctx, "NonExistent").Return(nil, errNotFound)

		result, err := svc.CreateRoleBinding(ctx, apiRB)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to resolve role")
		mockRole.AssertExpectations(t)
	})
}

func TestService_UpdateRoleBinding(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockRBUsecase := new(mockRoleBindingUsecase)
		mockRole := new(mockRoleUsecase)
		svc := newSvc(t, mockRBUsecase, mockRole)

		roleUID := uuid.New()
		role := &usermodel.Role{
			Metadata: usermodel.RoleMetadata{UID: roleUID},
			Spec:     usermodel.RoleSpec{DisplayName: "Viewer"},
		}

		apiRB := &v1.RoleBinding{
			Spec: v1.RoleBindingSpec{
				RoleRef:       v1.RoleBindingRoleRef{Kind: "Role", Name: "Viewer"},
				LabelSelector: map[string]string{"login-type": "github"},
			},
		}

		mockRole.On("GetRoleByName", ctx, "Viewer").Return(role, nil)
		mockRBUsecase.On("UpdateRoleBinding", ctx, "production", "viewer-binding",
			mock.MatchedBy(func(rb *usermodel.RoleBinding) bool {
				return rb.Spec.RoleRef.Name == "Viewer" && rb.Spec.LabelSelector["login-type"] == "github"
			}),
		).Return(newRB(), nil)

		result, err := svc.UpdateRoleBinding(ctx, "production", "viewer-binding", apiRB)

		require.NoError(t, err)
		assert.Equal(t, "production", result.Metadata.Namespace)
		mockRole.AssertExpectations(t)
		mockRBUsecase.AssertExpectations(t)
	})

	t.Run("role not found error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockRBUsecase := new(mockRoleBindingUsecase)
		mockRole := new(mockRoleUsecase)
		svc := newSvc(t, mockRBUsecase, mockRole)

		apiRB := &v1.RoleBinding{
			Spec: v1.RoleBindingSpec{
				RoleRef:       v1.RoleBindingRoleRef{Kind: "Role", Name: "Viewer"},
				LabelSelector: map[string]string{"login-type": "github"},
			},
		}

		mockRole.On("GetRoleByName", ctx, "Viewer").Return(nil, errNotFound)

		result, err := svc.UpdateRoleBinding(ctx, "production", "viewer-binding", apiRB)

		require.Error(t, err)
		assert.Nil(t, result)
		mockRole.AssertExpectations(t)
	})
}

func TestService_DeleteRoleBinding(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockRBUsecase := new(mockRoleBindingUsecase)
		mockRole := new(mockRoleUsecase)
		svc := newSvc(t, mockRBUsecase, mockRole)

		mockRBUsecase.On("DeleteRoleBinding", ctx, "production", "viewer-binding").Return(nil)

		err := svc.DeleteRoleBinding(ctx, "production", "viewer-binding")

		require.NoError(t, err)
		mockRBUsecase.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockRBUsecase := new(mockRoleBindingUsecase)
		mockRole := new(mockRoleUsecase)
		svc := newSvc(t, mockRBUsecase, mockRole)

		mockRBUsecase.On("DeleteRoleBinding", ctx, "production", "viewer-binding").Return(errMock)

		err := svc.DeleteRoleBinding(ctx, "production", "viewer-binding")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "delete role binding")
		mockRBUsecase.AssertExpectations(t)
	})
}
