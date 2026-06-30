package auth_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	applicationport "github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	authsvc "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/auth"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	usermodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

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

// stubRBACUsecase is a no-op userport.RBACUsecase whose SyncPolicies records that it ran.
type stubRBACUsecase struct {
	synced bool
}

func (s *stubRBACUsecase) SyncPolicies(context.Context) error {
	s.synced = true

	return nil
}

func (*stubRBACUsecase) CheckPermission(context.Context, uuid.UUID, string, string, string) (bool, error) {
	return true, nil
}

func (*stubRBACUsecase) GetUserPermissions(context.Context, uuid.UUID) ([]*usermodel.Permission, error) {
	return nil, nil
}

func (*stubRBACUsecase) GetEffectivePermissions(context.Context, uuid.UUID) ([]*usermodel.Permission, error) {
	return nil, nil
}

func newSvc(t *testing.T, user *mockUserUsecase, rbac *stubRBACUsecase) *authsvc.Service {
	t.Helper()

	base := testutil.NewBase(t)

	return authsvc.New(user, rbac, base.Logger)
}

func provisioning(email string) applicationport.LoginProvisioning {
	return applicationport.LoginProvisioning{
		Provider: usermodel.IdentityProviderGitHub,
		Username: "octocat",
		Email:    email,
		Groups:   []string{"acme"},
	}
}

func TestService_EnsureUserOnLogin(t *testing.T) {
	t.Parallel()

	t.Run("existing user is updated and policies synced", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockUser := new(mockUserUsecase)
		rbac := &stubRBACUsecase{}
		svc := newSvc(t, mockUser, rbac)

		existing := usermodel.NewUser("octo@example.com", "octocat")
		mockUser.On("GetUserByEmail", ctx, "octo@example.com").Return(existing, nil)
		mockUser.On("SaveUser", ctx, existing).Return(nil)

		svc.EnsureUserOnLogin(ctx, provisioning("octo@example.com"))

		mockUser.AssertCalled(t, "SaveUser", ctx, existing)
		require.True(t, rbac.synced)
		// The login-type label is synced onto the existing user.
		require.Equal(t, usermodel.IdentityProviderGitHub, existing.Metadata.Labels[usermodel.LabelLoginType])
	})

	t.Run("new user is created when none exists", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockUser := new(mockUserUsecase)
		rbac := &stubRBACUsecase{}
		svc := newSvc(t, mockUser, rbac)

		mockUser.On("GetUserByEmail", ctx, "new@example.com").Return(nil, model.ErrResourceNotExist)
		mockUser.On("GetUserByEmailIncludingDeleted", ctx, "new@example.com").
			Return(nil, model.ErrResourceNotExist)
		mockUser.On("SaveUser", ctx, mock.MatchedBy(func(u *usermodel.User) bool {
			return u.Spec.Email == "new@example.com"
		})).Return(nil)

		svc.EnsureUserOnLogin(ctx, provisioning("new@example.com"))

		mockUser.AssertExpectations(t)
		require.True(t, rbac.synced)
	})

	t.Run("soft-deleted user is not recreated", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockUser := new(mockUserUsecase)
		rbac := &stubRBACUsecase{}
		svc := newSvc(t, mockUser, rbac)

		deleted := usermodel.NewUser("gone@example.com", "gone")
		mockUser.On("GetUserByEmail", ctx, "gone@example.com").Return(nil, model.ErrResourceNotExist)
		mockUser.On("GetUserByEmailIncludingDeleted", ctx, "gone@example.com").Return(deleted, nil)

		svc.EnsureUserOnLogin(ctx, provisioning("gone@example.com"))

		mockUser.AssertNotCalled(t, "SaveUser", mock.Anything, mock.Anything)
		require.False(t, rbac.synced)
	})
}
