package userservice_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	usermodel "github.com/minuk-dev/opampcommander/internal/domain/user/model"
	userport "github.com/minuk-dev/opampcommander/internal/domain/user/port"
	userservice "github.com/minuk-dev/opampcommander/internal/domain/user/service"
)

var errUnexpectedType = errors.New("unexpected type")

var (
	errOrgRoleMappingPersistence = errors.New("org-role mapping persistence error")
)

// MockOrgRoleMappingPersistencePort is a mock implementation of OrgRoleMappingPersistencePort.
type MockOrgRoleMappingPersistencePort struct {
	mock.Mock
}

func (m *MockOrgRoleMappingPersistencePort) GetOrgRoleMapping(
	ctx context.Context,
	uid uuid.UUID,
) (*usermodel.OrgRoleMapping, error) {
	args := m.Called(ctx, uid)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	mapping, ok := args.Get(0).(*usermodel.OrgRoleMapping)
	if !ok {
		return nil, errUnexpectedType
	}

	return mapping, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockOrgRoleMappingPersistencePort) PutOrgRoleMapping(
	ctx context.Context,
	mapping *usermodel.OrgRoleMapping,
) (*usermodel.OrgRoleMapping, error) {
	args := m.Called(ctx, mapping)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	result, ok := args.Get(0).(*usermodel.OrgRoleMapping)
	if !ok {
		return nil, errUnexpectedType
	}

	return result, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockOrgRoleMappingPersistencePort) ListOrgRoleMappings(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*usermodel.OrgRoleMapping], error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, ok := args.Get(0).(*model.ListResponse[*usermodel.OrgRoleMapping])
	if !ok {
		return nil, errUnexpectedType
	}

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockOrgRoleMappingPersistencePort) ListOrgRoleMappingsByProvider(
	ctx context.Context,
	provider string,
) ([]*usermodel.OrgRoleMapping, error) {
	args := m.Called(ctx, provider)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	mappings, ok := args.Get(0).([]*usermodel.OrgRoleMapping)
	if !ok {
		return nil, errUnexpectedType
	}

	return mappings, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockOrgRoleMappingPersistencePort) DeleteOrgRoleMapping(
	ctx context.Context,
	uid uuid.UUID,
) error {
	args := m.Called(ctx, uid)

	return args.Error(0) //nolint:wrapcheck // mock error
}

// MockRolePersistencePort is a mock implementation of RolePersistencePort.
type MockRolePersistencePort struct {
	mock.Mock
}

func (m *MockRolePersistencePort) GetRole(ctx context.Context, uid uuid.UUID) (*usermodel.Role, error) {
	args := m.Called(ctx, uid)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	role, ok := args.Get(0).(*usermodel.Role)
	if !ok {
		return nil, errUnexpectedType
	}

	return role, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockRolePersistencePort) GetRoleByName(
	ctx context.Context,
	displayName string,
) (*usermodel.Role, error) {
	args := m.Called(ctx, displayName)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	role, ok := args.Get(0).(*usermodel.Role)
	if !ok {
		return nil, errUnexpectedType
	}

	return role, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockRolePersistencePort) PutRole(
	ctx context.Context,
	role *usermodel.Role,
) (*usermodel.Role, error) {
	args := m.Called(ctx, role)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	result, ok := args.Get(0).(*usermodel.Role)
	if !ok {
		return nil, errUnexpectedType
	}

	return result, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockRolePersistencePort) ListRoles(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*usermodel.Role], error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, ok := args.Get(0).(*model.ListResponse[*usermodel.Role])
	if !ok {
		return nil, errUnexpectedType
	}

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockRolePersistencePort) DeleteRole(ctx context.Context, uid uuid.UUID) error {
	args := m.Called(ctx, uid)

	return args.Error(0) //nolint:wrapcheck // mock error
}

// Ensure mocks implement the interfaces.
var _ userport.OrgRoleMappingPersistencePort = (*MockOrgRoleMappingPersistencePort)(nil)
var _ userport.RolePersistencePort = (*MockRolePersistencePort)(nil)

//nolint:maintidx // Table-driven test requires many test cases.
func TestOrgRoleMappingService_ResolveRolesForIdentity(t *testing.T) {
	t.Parallel()

	adminRoleID := uuid.New()
	viewerRoleID := uuid.New()

	adminRole := &usermodel.Role{
		Metadata: usermodel.RoleMetadata{UID: adminRoleID},
		Spec:     usermodel.RoleSpec{DisplayName: usermodel.RoleAdmin, IsBuiltIn: true},
	}
	viewerRole := &usermodel.Role{
		Metadata: usermodel.RoleMetadata{UID: viewerRoleID},
		Spec:     usermodel.RoleSpec{DisplayName: usermodel.RoleViewer, IsBuiltIn: true},
	}

	t.Run("Resolves admin role from GitHub org membership", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockMappingPort := new(MockOrgRoleMappingPersistencePort)
		mockRolePort := new(MockRolePersistencePort)
		logger := slog.Default()

		svc := userservice.NewOrgRoleMappingService(mockMappingPort, mockRolePort, logger)

		mappings := []*usermodel.OrgRoleMapping{
			usermodel.NewOrgRoleMapping("github", "my-company", "", adminRoleID),
			usermodel.NewOrgRoleMapping("github", "open-source-org", "", viewerRoleID),
		}

		identity := &usermodel.ExternalIdentity{
			Provider:       usermodel.IdentityProviderGitHub,
			ProviderUserID: "12345",
			Email:          "dev@company.com",
			Groups:         []string{"my-company"},
		}

		mockMappingPort.On("ListOrgRoleMappingsByProvider", ctx, "github").Return(mappings, nil)
		mockRolePort.On("GetRole", ctx, adminRoleID).Return(adminRole, nil)

		roles, err := svc.ResolveRolesForIdentity(ctx, identity)

		require.NoError(t, err)
		require.Len(t, roles, 1)
		assert.Equal(t, usermodel.RoleAdmin, roles[0].Spec.DisplayName)
		mockMappingPort.AssertExpectations(t)
		mockRolePort.AssertExpectations(t)
	})

	t.Run("Resolves multiple roles from multiple org memberships", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockMappingPort := new(MockOrgRoleMappingPersistencePort)
		mockRolePort := new(MockRolePersistencePort)
		logger := slog.Default()

		svc := userservice.NewOrgRoleMappingService(mockMappingPort, mockRolePort, logger)

		mappings := []*usermodel.OrgRoleMapping{
			usermodel.NewOrgRoleMapping("github", "my-company", "", adminRoleID),
			usermodel.NewOrgRoleMapping("github", "open-source-org", "", viewerRoleID),
		}

		identity := &usermodel.ExternalIdentity{
			Provider: usermodel.IdentityProviderGitHub,
			Groups:   []string{"my-company", "open-source-org"},
		}

		mockMappingPort.On("ListOrgRoleMappingsByProvider", ctx, "github").Return(mappings, nil)
		mockRolePort.On("GetRole", ctx, adminRoleID).Return(adminRole, nil)
		mockRolePort.On("GetRole", ctx, viewerRoleID).Return(viewerRole, nil)

		roles, err := svc.ResolveRolesForIdentity(ctx, identity)

		require.NoError(t, err)
		assert.Len(t, roles, 2)
		mockMappingPort.AssertExpectations(t)
		mockRolePort.AssertExpectations(t)
	})

	t.Run("Returns empty roles when no org matches", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockMappingPort := new(MockOrgRoleMappingPersistencePort)
		mockRolePort := new(MockRolePersistencePort)
		logger := slog.Default()

		svc := userservice.NewOrgRoleMappingService(mockMappingPort, mockRolePort, logger)

		mappings := []*usermodel.OrgRoleMapping{
			usermodel.NewOrgRoleMapping("github", "my-company", "", adminRoleID),
		}

		identity := &usermodel.ExternalIdentity{
			Provider: usermodel.IdentityProviderGitHub,
			Groups:   []string{"other-org"},
		}

		mockMappingPort.On("ListOrgRoleMappingsByProvider", ctx, "github").Return(mappings, nil)

		roles, err := svc.ResolveRolesForIdentity(ctx, identity)

		require.NoError(t, err)
		assert.Empty(t, roles)
		mockMappingPort.AssertExpectations(t)
		mockRolePort.AssertNotCalled(t, "GetRole")
	})

	t.Run("Returns empty roles for user with no groups", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockMappingPort := new(MockOrgRoleMappingPersistencePort)
		mockRolePort := new(MockRolePersistencePort)
		logger := slog.Default()

		svc := userservice.NewOrgRoleMappingService(mockMappingPort, mockRolePort, logger)

		mappings := []*usermodel.OrgRoleMapping{
			usermodel.NewOrgRoleMapping("github", "my-company", "", adminRoleID),
		}

		identity := &usermodel.ExternalIdentity{
			Provider: usermodel.IdentityProviderGitHub,
			Groups:   []string{},
		}

		mockMappingPort.On("ListOrgRoleMappingsByProvider", ctx, "github").Return(mappings, nil)

		roles, err := svc.ResolveRolesForIdentity(ctx, identity)

		require.NoError(t, err)
		assert.Empty(t, roles)
	})

	t.Run("Returns nil for nil identity", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockMappingPort := new(MockOrgRoleMappingPersistencePort)
		mockRolePort := new(MockRolePersistencePort)
		logger := slog.Default()

		svc := userservice.NewOrgRoleMappingService(mockMappingPort, mockRolePort, logger)

		roles, err := svc.ResolveRolesForIdentity(ctx, nil)

		require.NoError(t, err)
		assert.Nil(t, roles)
	})

	t.Run("Deduplicates roles when multiple orgs map to same role", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockMappingPort := new(MockOrgRoleMappingPersistencePort)
		mockRolePort := new(MockRolePersistencePort)
		logger := slog.Default()

		svc := userservice.NewOrgRoleMappingService(mockMappingPort, mockRolePort, logger)

		mappings := []*usermodel.OrgRoleMapping{
			usermodel.NewOrgRoleMapping("github", "org-a", "", adminRoleID),
			usermodel.NewOrgRoleMapping("github", "org-b", "", adminRoleID),
		}

		identity := &usermodel.ExternalIdentity{
			Provider: usermodel.IdentityProviderGitHub,
			Groups:   []string{"org-a", "org-b"},
		}

		mockMappingPort.On("ListOrgRoleMappingsByProvider", ctx, "github").Return(mappings, nil)
		mockRolePort.On("GetRole", ctx, adminRoleID).Return(adminRole, nil)

		roles, err := svc.ResolveRolesForIdentity(ctx, identity)

		require.NoError(t, err)
		assert.Len(t, roles, 1)
		assert.Equal(t, usermodel.RoleAdmin, roles[0].Spec.DisplayName)
	})

	t.Run("Skips roles that fail to resolve but continues", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockMappingPort := new(MockOrgRoleMappingPersistencePort)
		mockRolePort := new(MockRolePersistencePort)
		logger := slog.Default()

		svc := userservice.NewOrgRoleMappingService(mockMappingPort, mockRolePort, logger)

		brokenRoleID := uuid.New()

		mappings := []*usermodel.OrgRoleMapping{
			usermodel.NewOrgRoleMapping("github", "good-org", "", adminRoleID),
			usermodel.NewOrgRoleMapping("github", "bad-org", "", brokenRoleID),
		}

		identity := &usermodel.ExternalIdentity{
			Provider: usermodel.IdentityProviderGitHub,
			Groups:   []string{"good-org", "bad-org"},
		}

		mockMappingPort.On("ListOrgRoleMappingsByProvider", ctx, "github").Return(mappings, nil)
		mockRolePort.On("GetRole", ctx, adminRoleID).Return(adminRole, nil)
		mockRolePort.On("GetRole", ctx, brokenRoleID).Return(nil, port.ErrResourceNotExist)

		roles, err := svc.ResolveRolesForIdentity(ctx, identity)

		require.NoError(t, err)
		assert.Len(t, roles, 1)
		assert.Equal(t, usermodel.RoleAdmin, roles[0].Spec.DisplayName)
	})

	t.Run("Fails when listing mappings fails", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockMappingPort := new(MockOrgRoleMappingPersistencePort)
		mockRolePort := new(MockRolePersistencePort)
		logger := slog.Default()

		svc := userservice.NewOrgRoleMappingService(mockMappingPort, mockRolePort, logger)

		identity := &usermodel.ExternalIdentity{
			Provider: usermodel.IdentityProviderGitHub,
			Groups:   []string{"my-company"},
		}

		mockMappingPort.On("ListOrgRoleMappingsByProvider", ctx, "github").
			Return(nil, errOrgRoleMappingPersistence)

		roles, err := svc.ResolveRolesForIdentity(ctx, identity)

		require.Error(t, err)
		assert.Nil(t, roles)
		assert.Contains(t, err.Error(), "failed to list org-role mappings")
	})

	t.Run("Does not match mappings from different providers", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockMappingPort := new(MockOrgRoleMappingPersistencePort)
		mockRolePort := new(MockRolePersistencePort)
		logger := slog.Default()

		svc := userservice.NewOrgRoleMappingService(mockMappingPort, mockRolePort, logger)

		// The provider filter already happens at the persistence layer,
		// but verify no roles returned if the ListOrgRoleMappingsByProvider returns empty.
		mockMappingPort.On("ListOrgRoleMappingsByProvider", ctx, "google").
			Return([]*usermodel.OrgRoleMapping{}, nil)

		identity := &usermodel.ExternalIdentity{
			Provider: usermodel.IdentityProviderGoogle,
			Groups:   []string{"google-workspace-group"},
		}

		roles, err := svc.ResolveRolesForIdentity(ctx, identity)

		require.NoError(t, err)
		assert.Empty(t, roles)
	})
}

func TestOrgRoleMappingService_SaveOrgRoleMapping(t *testing.T) {
	t.Parallel()

	t.Run("Successfully saves mapping", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockMappingPort := new(MockOrgRoleMappingPersistencePort)
		mockRolePort := new(MockRolePersistencePort)
		logger := slog.Default()

		svc := userservice.NewOrgRoleMappingService(mockMappingPort, mockRolePort, logger)

		roleID := uuid.New()
		mapping := usermodel.NewOrgRoleMapping("github", "my-org", "", roleID)

		mockMappingPort.On("PutOrgRoleMapping", ctx, mapping).Return(mapping, nil)

		err := svc.SaveOrgRoleMapping(ctx, mapping)

		require.NoError(t, err)
		mockMappingPort.AssertExpectations(t)
	})

	t.Run("Persistence error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockMappingPort := new(MockOrgRoleMappingPersistencePort)
		mockRolePort := new(MockRolePersistencePort)
		logger := slog.Default()

		svc := userservice.NewOrgRoleMappingService(mockMappingPort, mockRolePort, logger)

		roleID := uuid.New()
		mapping := usermodel.NewOrgRoleMapping("github", "my-org", "", roleID)

		mockMappingPort.On("PutOrgRoleMapping", ctx, mapping).
			Return(nil, errOrgRoleMappingPersistence)

		err := svc.SaveOrgRoleMapping(ctx, mapping)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to save org-role mapping")
	})
}

func TestOrgRoleMappingService_DeleteOrgRoleMapping(t *testing.T) {
	t.Parallel()

	t.Run("Successfully deletes mapping", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockMappingPort := new(MockOrgRoleMappingPersistencePort)
		mockRolePort := new(MockRolePersistencePort)
		logger := slog.Default()

		svc := userservice.NewOrgRoleMappingService(mockMappingPort, mockRolePort, logger)

		uid := uuid.New()
		mockMappingPort.On("DeleteOrgRoleMapping", ctx, uid).Return(nil)

		err := svc.DeleteOrgRoleMapping(ctx, uid)

		require.NoError(t, err)
		mockMappingPort.AssertExpectations(t)
	})

	t.Run("Persistence error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockMappingPort := new(MockOrgRoleMappingPersistencePort)
		mockRolePort := new(MockRolePersistencePort)
		logger := slog.Default()

		svc := userservice.NewOrgRoleMappingService(mockMappingPort, mockRolePort, logger)

		uid := uuid.New()
		mockMappingPort.On("DeleteOrgRoleMapping", ctx, uid).Return(errOrgRoleMappingPersistence)

		err := svc.DeleteOrgRoleMapping(ctx, uid)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete org-role mapping")
	})
}

// Verify OrgRoleMappingService implements OrgRoleMappingUsecase interface.
var _ userport.OrgRoleMappingUsecase = (*userservice.OrgRoleMappingService)(nil)
