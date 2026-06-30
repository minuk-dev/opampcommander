package user_test

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
	usersvc "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/user"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	usermodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

var errMock = errors.New("mock error")

// mockUserUsecase is a mock implementation of userport.UserUsecase.
type mockUserUsecase struct {
	mock.Mock
}

func (m *mockUserUsecase) GetUser(
	ctx context.Context, uid uuid.UUID, options *model.GetOptions,
) (*usermodel.User, error) {
	args := m.Called(ctx, uid, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	user, _ := args.Get(0).(*usermodel.User)

	return user, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockUserUsecase) GetUserByEmail(ctx context.Context, email string) (*usermodel.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	user, _ := args.Get(0).(*usermodel.User)

	return user, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockUserUsecase) GetUserByEmailIncludingDeleted(
	ctx context.Context, email string,
) (*usermodel.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	user, _ := args.Get(0).(*usermodel.User)

	return user, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockUserUsecase) GetUserByUsername(ctx context.Context, username string) (*usermodel.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	user, _ := args.Get(0).(*usermodel.User)

	return user, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockUserUsecase) ListUsers(
	ctx context.Context, options *model.ListOptions,
) (*model.ListResponse[*usermodel.User], error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, _ := args.Get(0).(*model.ListResponse[*usermodel.User])

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockUserUsecase) SaveUser(ctx context.Context, user *usermodel.User) error {
	args := m.Called(ctx, user)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func (m *mockUserUsecase) DeleteUser(ctx context.Context, uid uuid.UUID) error {
	args := m.Called(ctx, uid)

	return args.Error(0) //nolint:wrapcheck // mock error
}

// stubRBACUsecase is a no-op userport.RBACUsecase.
type stubRBACUsecase struct{}

func (*stubRBACUsecase) SyncPolicies(context.Context) error { return nil }

func (*stubRBACUsecase) CheckPermission(context.Context, uuid.UUID, string, string, string) (bool, error) {
	return true, nil
}

func (*stubRBACUsecase) GetUserPermissions(context.Context, uuid.UUID) ([]*usermodel.Permission, error) {
	return nil, nil
}

func (*stubRBACUsecase) GetEffectivePermissions(context.Context, uuid.UUID) ([]*usermodel.Permission, error) {
	return nil, nil
}

// newSvc builds the user service. roleUsecase, roleBindingPersistencePort, rbacEnforcerPort and
// passwordHasher are unused by the CRUD methods under test, so they are left nil.
func newSvc(t *testing.T, user *mockUserUsecase) *usersvc.Service {
	t.Helper()

	base := testutil.NewBase(t)

	return usersvc.New(user, nil, nil, nil, &stubRBACUsecase{}, nil, base.Logger)
}

func TestService_GetUser(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockUser := new(mockUserUsecase)
		svc := newSvc(t, mockUser)

		uid := uuid.New()
		mockUser.On("GetUser", ctx, uid, (*model.GetOptions)(nil)).
			Return(usermodel.NewUser("a@example.com", "alice"), nil)

		result, err := svc.GetUser(ctx, uid, nil)

		require.NoError(t, err)
		assert.Equal(t, v1.UserKind, result.Kind)
		assert.Equal(t, "a@example.com", result.Spec.Email)
		mockUser.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockUser := new(mockUserUsecase)
		svc := newSvc(t, mockUser)

		uid := uuid.New()
		mockUser.On("GetUser", ctx, uid, (*model.GetOptions)(nil)).Return(nil, errMock)

		result, err := svc.GetUser(ctx, uid, nil)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to get user")
		mockUser.AssertExpectations(t)
	})
}

func TestService_GetUserByEmail(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockUser := new(mockUserUsecase)
		svc := newSvc(t, mockUser)

		mockUser.On("GetUserByEmail", ctx, "a@example.com").
			Return(usermodel.NewUser("a@example.com", "alice"), nil)

		result, err := svc.GetUserByEmail(ctx, "a@example.com")

		require.NoError(t, err)
		assert.Equal(t, "a@example.com", result.Spec.Email)
		mockUser.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockUser := new(mockUserUsecase)
		svc := newSvc(t, mockUser)

		mockUser.On("GetUserByEmail", ctx, "missing@example.com").Return(nil, errMock)

		result, err := svc.GetUserByEmail(ctx, "missing@example.com")

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to get user by email")
		mockUser.AssertExpectations(t)
	})
}

func TestService_ListUsers(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockUser := new(mockUserUsecase)
		svc := newSvc(t, mockUser)

		opts := &applicationport.ListOptions{Limit: 10}
		resp := &model.ListResponse[*usermodel.User]{
			Items:    []*usermodel.User{usermodel.NewUser("a@example.com", "alice")},
			Continue: "next",
		}
		mockUser.On("ListUsers", ctx, opts.ToDomain()).Return(resp, nil)

		result, err := svc.ListUsers(ctx, opts)

		require.NoError(t, err)
		assert.Equal(t, v1.UserKind, result.Kind)
		assert.Len(t, result.Items, 1)
		assert.Equal(t, "next", result.Metadata.Continue)
		mockUser.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockUser := new(mockUserUsecase)
		svc := newSvc(t, mockUser)

		opts := &applicationport.ListOptions{Limit: 10}
		mockUser.On("ListUsers", ctx, opts.ToDomain()).Return(nil, errMock)

		result, err := svc.ListUsers(ctx, opts)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to list users")
		mockUser.AssertExpectations(t)
	})
}

func TestService_CreateUser(t *testing.T) {
	t.Parallel()

	t.Run("success without password", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockUser := new(mockUserUsecase)
		svc := newSvc(t, mockUser)

		apiUser := &v1.User{
			Kind:       v1.UserKind,
			APIVersion: v1.APIVersion,
			Spec:       v1.UserSpec{Email: "a@example.com", Username: "alice"},
		}
		mockUser.On("SaveUser", ctx, mock.MatchedBy(func(u *usermodel.User) bool {
			return u.Spec.Email == "a@example.com"
		})).Return(nil)

		result, err := svc.CreateUser(ctx, apiUser)

		require.NoError(t, err)
		assert.Equal(t, "a@example.com", result.Spec.Email)
		mockUser.AssertExpectations(t)
	})

	t.Run("save error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockUser := new(mockUserUsecase)
		svc := newSvc(t, mockUser)

		apiUser := &v1.User{Spec: v1.UserSpec{Email: "a@example.com", Username: "alice"}}
		mockUser.On("SaveUser", ctx, mock.Anything).Return(errMock)

		result, err := svc.CreateUser(ctx, apiUser)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to create user")
		mockUser.AssertExpectations(t)
	})
}

func TestService_DeleteUser(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockUser := new(mockUserUsecase)
		svc := newSvc(t, mockUser)

		uid := uuid.New()
		mockUser.On("DeleteUser", ctx, uid).Return(nil)

		err := svc.DeleteUser(ctx, uid)

		require.NoError(t, err)
		mockUser.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockUser := new(mockUserUsecase)
		svc := newSvc(t, mockUser)

		uid := uuid.New()
		mockUser.On("DeleteUser", ctx, uid).Return(errMock)

		err := svc.DeleteUser(ctx, uid)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete user")
		mockUser.AssertExpectations(t)
	})
}
