package service_test

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
	"github.com/minuk-dev/opampcommander/internal/domain/service"
)

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
) (*model.OrgRoleMapping, error) {
	args := m.Called(ctx, uid)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	mapping, ok := args.Get(0).(*model.OrgRoleMapping)
	if !ok {
		return nil, errUnexpectedType
	}

	return mapping, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockOrgRoleMappingPersistencePort) PutOrgRoleMapping(
	ctx context.Context,
	mapping *model.OrgRoleMapping,
) (*model.OrgRoleMapping, error) {
	args := m.Called(ctx, mapping)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	result, ok := args.Get(0).(*model.OrgRoleMapping)
	if !ok {
		return nil, errUnexpectedType
	}

	return result, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockOrgRoleMappingPersistencePort) ListOrgRoleMappings(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*model.OrgRoleMapping], error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, ok := args.Get(0).(*model.ListResponse[*model.OrgRoleMapping])
	if !ok {
		return nil, errUnexpectedType
	}

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockOrgRoleMappingPersistencePort) ListOrgRoleMappingsByProvider(
	ctx context.Context,
	provider string,
) ([]*model.OrgRoleMapping, error) {
	args := m.Called(ctx, provider)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	mappings, ok := args.Get(0).([]*model.OrgRoleMapping)
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

func (m *MockRolePersistencePort) GetRole(ctx context.Context, uid uuid.UUID) (*model.Role, error) {
	args := m.Called(ctx, uid)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	role, ok := args.Get(0).(*model.Role)
	if !ok {
		return nil, errUnexpectedType
	}

	return role, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockRolePersistencePort) GetRoleByName(
	ctx context.Context,
	displayName string,
) (*model.Role, error) {
	args := m.Called(ctx, displayName)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	role, ok := args.Get(0).(*model.Role)
	if !ok {
		return nil, errUnexpectedType
	}

	return role, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockRolePersistencePort) PutRole(
	ctx context.Context,
	role *model.Role,
) (*model.Role, error) {
	args := m.Called(ctx, role)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	result, ok := args.Get(0).(*model.Role)
	if !ok {
		return nil, errUnexpectedType
	}

	return result, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockRolePersistencePort) ListRoles(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*model.Role], error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, ok := args.Get(0).(*model.ListResponse[*model.Role])
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
var _ port.OrgRoleMappingPersistencePort = (*MockOrgRoleMappingPersistencePort)(nil)
var _ port.RolePersistencePort = (*MockRolePersistencePort)(nil)

func TestOrgRoleMappingService_ResolveRolesForIdentity(t *testing.T) {
	t.Parallel()

	adminRoleID := uuid.New()
	viewerRoleID := uuid.New()

	adminRole := &model.Role{
		Metadata: model.RoleMetadata{UID: adminRoleID},
		Spec:     model.RoleSpec{DisplayName: model.RoleAdmin, IsBuiltIn: true},
	}
	viewerRole := &model.Role{
		Metadata: model.RoleMetadata{UID: viewerRoleID},
		Spec:     model.RoleSpec{DisplayName: model.RoleViewer, IsBuiltIn: true},
	}

	t.Run("Resolves admin role from GitHub org membership", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		mockMappingPort := new(MockOrgRoleMappingPersistencePort)
		mockRolePort := new(MockRolePersistencePort)
		logger := slog.Default()

		svc := service.NewOrgRoleMappingService(mockMappingPort, mockRolePort, logger)

		mappings := []*model.OrgRoleMapping{
			model.NewOrgRoleMapping("github", "my-company", "", adminRoleID),
			model.NewOrgRoleMapping("github", "open-source-org", "", viewerRoleID),
		}

		identity := &model.ExternalIdentity{
			Provider:       model.IdentityProviderGitHub,
			ProviderUserID: "12345",
			Email:          "dev@company.com",
			Groups:         []string{"my-company"},
		}

		mockMappingPort.On("ListOrgRoleMappingsByProvider", ctx, "github").Return(mappings, nil)
		mockRolePort.On("GetRole", ctx, adminRoleID).Return(adminRole, nil)

		roles, err := svc.ResolveRolesForIdentity(ctx, identity)

		require.NoError(t, err)
		require.Len(t, roles, 1)
		assert.Equal(t, model.RoleAdmin, roles[0].Spec.DisplayName)
		mockMappingPort.AssertExpectations(t)
		mockRolePort.AssertExpectations(t)
	})

	t.Run("Resolves multiple roles from multiple org memberships", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		mockMappingPort := new(MockOrgRoleMappingPersistencePort)
		mockRolePort := new(MockRolePersistencePort)
		logger := slog.Default()

		svc := service.NewOrgRoleMappingService(mockMappingPort, mockRolePort, logger)

		mappings := []*model.OrgRoleMapping{
			model.NewOrgRoleMapping("github", "my-company", "", adminRoleID),
			model.NewOrgRoleMapping("github", "open-source-org", "", viewerRoleID),
		}

		identity := &model.ExternalIdentity{
			Provider: model.IdentityProviderGitHub,
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

		ctx := context.Background()
		mockMappingPort := new(MockOrgRoleMappingPersistencePort)
		mockRolePort := new(MockRolePersistencePort)
		logger := slog.Default()

		svc := service.NewOrgRoleMappingService(mockMappingPort, mockRolePort, logger)

		mappings := []*model.OrgRoleMapping{
			model.NewOrgRoleMapping("github", "my-company", "", adminRoleID),
		}

		identity := &model.ExternalIdentity{
			Provider: model.IdentityProviderGitHub,
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

		ctx := context.Background()
		mockMappingPort := new(MockOrgRoleMappingPersistencePort)
		mockRolePort := new(MockRolePersistencePort)
		logger := slog.Default()

		svc := service.NewOrgRoleMappingService(mockMappingPort, mockRolePort, logger)

		mappings := []*model.OrgRoleMapping{
			model.NewOrgRoleMapping("github", "my-company", "", adminRoleID),
		}

		identity := &model.ExternalIdentity{
			Provider: model.IdentityProviderGitHub,
			Groups:   []string{},
		}

		mockMappingPort.On("ListOrgRoleMappingsByProvider", ctx, "github").Return(mappings, nil)

		roles, err := svc.ResolveRolesForIdentity(ctx, identity)

		require.NoError(t, err)
		assert.Empty(t, roles)
	})

	t.Run("Returns nil for nil identity", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		mockMappingPort := new(MockOrgRoleMappingPersistencePort)
		mockRolePort := new(MockRolePersistencePort)
		logger := slog.Default()

		svc := service.NewOrgRoleMappingService(mockMappingPort, mockRolePort, logger)

		roles, err := svc.ResolveRolesForIdentity(ctx, nil)

		require.NoError(t, err)
		assert.Nil(t, roles)
	})

	t.Run("Deduplicates roles when multiple orgs map to same role", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		mockMappingPort := new(MockOrgRoleMappingPersistencePort)
		mockRolePort := new(MockRolePersistencePort)
		logger := slog.Default()

		svc := service.NewOrgRoleMappingService(mockMappingPort, mockRolePort, logger)

		mappings := []*model.OrgRoleMapping{
			model.NewOrgRoleMapping("github", "org-a", "", adminRoleID),
			model.NewOrgRoleMapping("github", "org-b", "", adminRoleID),
		}

		identity := &model.ExternalIdentity{
			Provider: model.IdentityProviderGitHub,
			Groups:   []string{"org-a", "org-b"},
		}

		mockMappingPort.On("ListOrgRoleMappingsByProvider", ctx, "github").Return(mappings, nil)
		mockRolePort.On("GetRole", ctx, adminRoleID).Return(adminRole, nil)

		roles, err := svc.ResolveRolesForIdentity(ctx, identity)

		require.NoError(t, err)
		assert.Len(t, roles, 1)
		assert.Equal(t, model.RoleAdmin, roles[0].Spec.DisplayName)
	})

	t.Run("Skips roles that fail to resolve but continues", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		mockMappingPort := new(MockOrgRoleMappingPersistencePort)
		mockRolePort := new(MockRolePersistencePort)
		logger := slog.Default()

		svc := service.NewOrgRoleMappingService(mockMappingPort, mockRolePort, logger)

		brokenRoleID := uuid.New()

		mappings := []*model.OrgRoleMapping{
			model.NewOrgRoleMapping("github", "good-org", "", adminRoleID),
			model.NewOrgRoleMapping("github", "bad-org", "", brokenRoleID),
		}

		identity := &model.ExternalIdentity{
			Provider: model.IdentityProviderGitHub,
			Groups:   []string{"good-org", "bad-org"},
		}

		mockMappingPort.On("ListOrgRoleMappingsByProvider", ctx, "github").Return(mappings, nil)
		mockRolePort.On("GetRole", ctx, adminRoleID).Return(adminRole, nil)
		mockRolePort.On("GetRole", ctx, brokenRoleID).Return(nil, port.ErrResourceNotExist)

		roles, err := svc.ResolveRolesForIdentity(ctx, identity)

		require.NoError(t, err)
		assert.Len(t, roles, 1)
		assert.Equal(t, model.RoleAdmin, roles[0].Spec.DisplayName)
	})

	t.Run("Fails when listing mappings fails", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		mockMappingPort := new(MockOrgRoleMappingPersistencePort)
		mockRolePort := new(MockRolePersistencePort)
		logger := slog.Default()

		svc := service.NewOrgRoleMappingService(mockMappingPort, mockRolePort, logger)

		identity := &model.ExternalIdentity{
			Provider: model.IdentityProviderGitHub,
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

		ctx := context.Background()
		mockMappingPort := new(MockOrgRoleMappingPersistencePort)
		mockRolePort := new(MockRolePersistencePort)
		logger := slog.Default()

		svc := service.NewOrgRoleMappingService(mockMappingPort, mockRolePort, logger)

		// The provider filter already happens at the persistence layer,
		// but verify no roles returned if the ListOrgRoleMappingsByProvider returns empty.
		mockMappingPort.On("ListOrgRoleMappingsByProvider", ctx, "google").
			Return([]*model.OrgRoleMapping{}, nil)

		identity := &model.ExternalIdentity{
			Provider: model.IdentityProviderGoogle,
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

		ctx := context.Background()
		mockMappingPort := new(MockOrgRoleMappingPersistencePort)
		mockRolePort := new(MockRolePersistencePort)
		logger := slog.Default()

		svc := service.NewOrgRoleMappingService(mockMappingPort, mockRolePort, logger)

		roleID := uuid.New()
		mapping := model.NewOrgRoleMapping("github", "my-org", "", roleID)

		mockMappingPort.On("PutOrgRoleMapping", ctx, mapping).Return(mapping, nil)

		err := svc.SaveOrgRoleMapping(ctx, mapping)

		require.NoError(t, err)
		mockMappingPort.AssertExpectations(t)
	})

	t.Run("Persistence error", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		mockMappingPort := new(MockOrgRoleMappingPersistencePort)
		mockRolePort := new(MockRolePersistencePort)
		logger := slog.Default()

		svc := service.NewOrgRoleMappingService(mockMappingPort, mockRolePort, logger)

		roleID := uuid.New()
		mapping := model.NewOrgRoleMapping("github", "my-org", "", roleID)

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

		ctx := context.Background()
		mockMappingPort := new(MockOrgRoleMappingPersistencePort)
		mockRolePort := new(MockRolePersistencePort)
		logger := slog.Default()

		svc := service.NewOrgRoleMappingService(mockMappingPort, mockRolePort, logger)

		uid := uuid.New()
		mockMappingPort.On("DeleteOrgRoleMapping", ctx, uid).Return(nil)

		err := svc.DeleteOrgRoleMapping(ctx, uid)

		require.NoError(t, err)
		mockMappingPort.AssertExpectations(t)
	})

	t.Run("Persistence error", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		mockMappingPort := new(MockOrgRoleMappingPersistencePort)
		mockRolePort := new(MockRolePersistencePort)
		logger := slog.Default()

		svc := service.NewOrgRoleMappingService(mockMappingPort, mockRolePort, logger)

		uid := uuid.New()
		mockMappingPort.On("DeleteOrgRoleMapping", ctx, uid).Return(errOrgRoleMappingPersistence)

		err := svc.DeleteOrgRoleMapping(ctx, uid)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete org-role mapping")
	})
}

// Verify OrgRoleMappingService implements OrgRoleMappingUsecase interface.
var _ port.OrgRoleMappingUsecase = (*service.OrgRoleMappingService)(nil)
