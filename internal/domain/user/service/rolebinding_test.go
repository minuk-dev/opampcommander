package userservice_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	usermodel "github.com/minuk-dev/opampcommander/internal/domain/user/model"
	userport "github.com/minuk-dev/opampcommander/internal/domain/user/port"
	userservice "github.com/minuk-dev/opampcommander/internal/domain/user/service"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

var errRoleBindingPersistence = errors.New("role binding persistence error")

// mockRoleBindingPersistencePort is a mock implementation of RoleBindingPersistencePort.
type mockRoleBindingPersistencePort struct {
	mock.Mock
}

var _ userport.RoleBindingPersistencePort = (*mockRoleBindingPersistencePort)(nil)

func (m *mockRoleBindingPersistencePort) GetRoleBinding(
	ctx context.Context,
	namespace, name string,
) (*usermodel.RoleBinding, error) {
	args := m.Called(ctx, namespace, name)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	rb, ok := args.Get(0).(*usermodel.RoleBinding)
	if !ok {
		return nil, errUnexpectedType
	}

	return rb, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockRoleBindingPersistencePort) PutRoleBinding(
	ctx context.Context,
	rb *usermodel.RoleBinding,
) (*usermodel.RoleBinding, error) {
	args := m.Called(ctx, rb)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	result, ok := args.Get(0).(*usermodel.RoleBinding)
	if !ok {
		return nil, errUnexpectedType
	}

	return result, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockRoleBindingPersistencePort) ListRoleBindings(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*usermodel.RoleBinding], error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, ok := args.Get(0).(*model.ListResponse[*usermodel.RoleBinding])
	if !ok {
		return nil, errUnexpectedType
	}

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockRoleBindingPersistencePort) DeleteRoleBinding(
	ctx context.Context,
	namespace, name string,
) error {
	args := m.Called(ctx, namespace, name)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func TestRoleBindingService_GetRoleBinding(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		base := testutil.NewBase(t)
		mockPort := new(mockRoleBindingPersistencePort)
		svc := userservice.NewRoleBindingService(mockPort, base.Logger)

		rb := usermodel.NewRoleBinding("production", "viewer-binding",
			usermodel.RoleRef{Kind: "Role", Name: "Viewer", UID: uuid.New()},
			usermodel.Subject{Kind: "User", Name: "alice@example.com", UID: uuid.New()},
		)

		mockPort.On("GetRoleBinding", ctx, "production", "viewer-binding").Return(rb, nil)

		result, err := svc.GetRoleBinding(ctx, "production", "viewer-binding")

		require.NoError(t, err)
		assert.Equal(t, rb, result)
		mockPort.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		base := testutil.NewBase(t)
		mockPort := new(mockRoleBindingPersistencePort)
		svc := userservice.NewRoleBindingService(mockPort, base.Logger)

		mockPort.On("GetRoleBinding", ctx, "production", "viewer-binding").
			Return(nil, errRoleBindingPersistence)

		result, err := svc.GetRoleBinding(ctx, "production", "viewer-binding")

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to get role binding from persistence")
		mockPort.AssertExpectations(t)
	})
}

func TestRoleBindingService_ListRoleBindings(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		base := testutil.NewBase(t)
		mockPort := new(mockRoleBindingPersistencePort)
		svc := userservice.NewRoleBindingService(mockPort, base.Logger)

		rb := usermodel.NewRoleBinding("production", "viewer-binding",
			usermodel.RoleRef{Kind: "Role", Name: "Viewer", UID: uuid.New()},
			usermodel.Subject{Kind: "User", Name: "alice@example.com", UID: uuid.New()},
		)
		resp := &model.ListResponse[*usermodel.RoleBinding]{
			Items: []*usermodel.RoleBinding{rb},
		}

		opts := &model.ListOptions{Limit: 10}
		mockPort.On("ListRoleBindings", ctx, opts).Return(resp, nil)

		result, err := svc.ListRoleBindings(ctx, opts)

		require.NoError(t, err)
		assert.Len(t, result.Items, 1)
		mockPort.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		base := testutil.NewBase(t)
		mockPort := new(mockRoleBindingPersistencePort)
		svc := userservice.NewRoleBindingService(mockPort, base.Logger)

		opts := &model.ListOptions{Limit: 10}
		mockPort.On("ListRoleBindings", ctx, opts).Return(nil, errRoleBindingPersistence)

		result, err := svc.ListRoleBindings(ctx, opts)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to list role bindings from persistence")
		mockPort.AssertExpectations(t)
	})
}

func TestRoleBindingService_CreateRoleBinding(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		base := testutil.NewBase(t)
		mockPort := new(mockRoleBindingPersistencePort)
		svc := userservice.NewRoleBindingService(mockPort, base.Logger)

		rb := usermodel.NewRoleBinding("production", "viewer-binding",
			usermodel.RoleRef{Kind: "Role", Name: "Viewer", UID: uuid.New()},
			usermodel.Subject{Kind: "User", Name: "alice@example.com", UID: uuid.New()},
		)

		mockPort.On("PutRoleBinding", ctx, rb).Return(rb, nil)

		result, err := svc.CreateRoleBinding(ctx, rb)

		require.NoError(t, err)
		assert.Equal(t, rb, result)
		mockPort.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		base := testutil.NewBase(t)
		mockPort := new(mockRoleBindingPersistencePort)
		svc := userservice.NewRoleBindingService(mockPort, base.Logger)

		rb := usermodel.NewRoleBinding("production", "viewer-binding",
			usermodel.RoleRef{Kind: "Role", Name: "Viewer", UID: uuid.New()},
			usermodel.Subject{Kind: "User", Name: "alice@example.com", UID: uuid.New()},
		)

		mockPort.On("PutRoleBinding", ctx, rb).Return(nil, errRoleBindingPersistence)

		result, err := svc.CreateRoleBinding(ctx, rb)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to create role binding in persistence")
		mockPort.AssertExpectations(t)
	})
}

func TestRoleBindingService_UpdateRoleBinding(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		base := testutil.NewBase(t)
		mockPort := new(mockRoleBindingPersistencePort)
		svc := userservice.NewRoleBindingService(mockPort, base.Logger)

		rb := usermodel.NewRoleBinding("production", "viewer-binding",
			usermodel.RoleRef{Kind: "Role", Name: "Viewer", UID: uuid.New()},
			usermodel.Subject{Kind: "User", Name: "alice@example.com", UID: uuid.New()},
		)
		beforeUpdate := rb.Metadata.UpdatedAt

		mockPort.On("PutRoleBinding", ctx, mock.MatchedBy(func(r *usermodel.RoleBinding) bool {
			return r.Metadata.Namespace == "production" &&
				r.Metadata.Name == "viewer-binding" &&
				r.Metadata.UpdatedAt.After(beforeUpdate)
		})).Return(rb, nil)

		// Small sleep to ensure time difference
		time.Sleep(time.Millisecond)

		result, err := svc.UpdateRoleBinding(ctx, "production", "viewer-binding", rb)

		require.NoError(t, err)
		assert.Equal(t, rb, result)
		mockPort.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		base := testutil.NewBase(t)
		mockPort := new(mockRoleBindingPersistencePort)
		svc := userservice.NewRoleBindingService(mockPort, base.Logger)

		rb := usermodel.NewRoleBinding("production", "viewer-binding",
			usermodel.RoleRef{Kind: "Role", Name: "Viewer", UID: uuid.New()},
			usermodel.Subject{Kind: "User", Name: "alice@example.com", UID: uuid.New()},
		)

		mockPort.On("PutRoleBinding", ctx, mock.Anything).Return(nil, errRoleBindingPersistence)

		result, err := svc.UpdateRoleBinding(ctx, "production", "viewer-binding", rb)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to update role binding in persistence")
		mockPort.AssertExpectations(t)
	})
}

func TestRoleBindingService_DeleteRoleBinding(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		base := testutil.NewBase(t)
		mockPort := new(mockRoleBindingPersistencePort)
		svc := userservice.NewRoleBindingService(mockPort, base.Logger)

		mockPort.On("DeleteRoleBinding", ctx, "production", "viewer-binding").Return(nil)

		err := svc.DeleteRoleBinding(ctx, "production", "viewer-binding")

		require.NoError(t, err)
		mockPort.AssertExpectations(t)
	})

	t.Run("delete error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		base := testutil.NewBase(t)
		mockPort := new(mockRoleBindingPersistencePort)
		svc := userservice.NewRoleBindingService(mockPort, base.Logger)

		mockPort.On("DeleteRoleBinding", ctx, "production", "viewer-binding").
			Return(errRoleBindingPersistence)

		err := svc.DeleteRoleBinding(ctx, "production", "viewer-binding")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete role binding")
		mockPort.AssertExpectations(t)
	})
}

// Verify RoleBindingService implements RoleBindingUsecase interface.
var _ userport.RoleBindingUsecase = (*userservice.RoleBindingService)(nil)
