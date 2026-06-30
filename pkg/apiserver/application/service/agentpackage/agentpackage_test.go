package agentpackage_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	applicationport "github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	agentpackagesvc "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/agentpackage"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

var errMock = errors.New("mock error")

// mockAgentPackageUsecase is a mock implementation of agentport.AgentPackageUsecase.
type mockAgentPackageUsecase struct {
	mock.Mock
}

func (m *mockAgentPackageUsecase) GetAgentPackage(
	ctx context.Context, namespace, name string, options *model.GetOptions,
) (*agentmodel.AgentPackage, error) {
	args := m.Called(ctx, namespace, name, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	pkg, ok := args.Get(0).(*agentmodel.AgentPackage)
	if !ok {
		return nil, errMock
	}

	return pkg, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockAgentPackageUsecase) ListAgentPackages(
	ctx context.Context, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.AgentPackage], error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, ok := args.Get(0).(*model.ListResponse[*agentmodel.AgentPackage])
	if !ok {
		return nil, errMock
	}

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockAgentPackageUsecase) SaveAgentPackage(
	ctx context.Context, agentPackage *agentmodel.AgentPackage,
) (*agentmodel.AgentPackage, error) {
	args := m.Called(ctx, agentPackage)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	pkg, ok := args.Get(0).(*agentmodel.AgentPackage)
	if !ok {
		return nil, errMock
	}

	return pkg, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockAgentPackageUsecase) CreateAgentPackage(
	ctx context.Context, agentPackage *agentmodel.AgentPackage, actor string,
) (*agentmodel.AgentPackage, error) {
	args := m.Called(ctx, agentPackage, actor)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	pkg, ok := args.Get(0).(*agentmodel.AgentPackage)
	if !ok {
		return nil, errMock
	}

	return pkg, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockAgentPackageUsecase) UpdateAgentPackage(
	ctx context.Context, namespace, name string, agentPackage *agentmodel.AgentPackage,
) (*agentmodel.AgentPackage, error) {
	args := m.Called(ctx, namespace, name, agentPackage)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	pkg, ok := args.Get(0).(*agentmodel.AgentPackage)
	if !ok {
		return nil, errMock
	}

	return pkg, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockAgentPackageUsecase) DeleteAgentPackage(
	ctx context.Context, namespace, name string, deletedAt time.Time, deletedBy string,
) error {
	args := m.Called(ctx, namespace, name, deletedAt, deletedBy)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func newSvc(t *testing.T, pkg *mockAgentPackageUsecase) *agentpackagesvc.Service {
	t.Helper()

	base := testutil.NewBase(t)

	return agentpackagesvc.NewAgentPackageService(pkg, base.Logger)
}

func newPkg() *agentmodel.AgentPackage {
	return &agentmodel.AgentPackage{
		Metadata: agentmodel.AgentPackageMetadata{Namespace: "default", Name: "pkg-1"},
	}
}

func apiPkg() *v1.AgentPackage {
	return &v1.AgentPackage{
		Kind:       v1.AgentPackageKind,
		APIVersion: v1.APIVersion,
		Metadata:   v1.AgentPackageMetadata{Namespace: "default", Name: "pkg-1"},
	}
}

func TestService_GetAgentPackage(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockPkg := new(mockAgentPackageUsecase)
		svc := newSvc(t, mockPkg)

		mockPkg.On("GetAgentPackage", ctx, "default", "pkg-1", (*model.GetOptions)(nil)).
			Return(newPkg(), nil)

		result, err := svc.GetAgentPackage(ctx, "default", "pkg-1", nil)

		require.NoError(t, err)
		assert.Equal(t, "default", result.Metadata.Namespace)
		assert.Equal(t, "pkg-1", result.Metadata.Name)
		mockPkg.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockPkg := new(mockAgentPackageUsecase)
		svc := newSvc(t, mockPkg)

		mockPkg.On("GetAgentPackage", ctx, "default", "missing", (*model.GetOptions)(nil)).Return(nil, errMock)

		result, err := svc.GetAgentPackage(ctx, "default", "missing", nil)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "get agent package")
		mockPkg.AssertExpectations(t)
	})
}

func TestService_ListAgentPackages(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockPkg := new(mockAgentPackageUsecase)
		svc := newSvc(t, mockPkg)

		opts := &applicationport.ListOptions{Limit: 10}
		resp := &model.ListResponse[*agentmodel.AgentPackage]{
			Items:    []*agentmodel.AgentPackage{newPkg()},
			Continue: "next",
		}
		mockPkg.On("ListAgentPackages", ctx, opts.ToDomain()).Return(resp, nil)

		result, err := svc.ListAgentPackages(ctx, opts)

		require.NoError(t, err)
		assert.Equal(t, v1.AgentPackageKind, result.Kind)
		assert.Len(t, result.Items, 1)
		assert.Equal(t, "next", result.Metadata.Continue)
		mockPkg.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockPkg := new(mockAgentPackageUsecase)
		svc := newSvc(t, mockPkg)

		opts := &applicationport.ListOptions{Limit: 10}
		mockPkg.On("ListAgentPackages", ctx, opts.ToDomain()).Return(nil, errMock)

		result, err := svc.ListAgentPackages(ctx, opts)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "list agent packages")
		mockPkg.AssertExpectations(t)
	})
}

func TestService_CreateAgentPackage(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockPkg := new(mockAgentPackageUsecase)
		svc := newSvc(t, mockPkg)

		mockPkg.On("CreateAgentPackage", ctx, mock.MatchedBy(func(p *agentmodel.AgentPackage) bool {
			return p.Metadata.Name == "pkg-1"
		}), mock.AnythingOfType("string")).Return(newPkg(), nil)

		result, err := svc.CreateAgentPackage(ctx, apiPkg())

		require.NoError(t, err)
		assert.Equal(t, "pkg-1", result.Metadata.Name)
		mockPkg.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockPkg := new(mockAgentPackageUsecase)
		svc := newSvc(t, mockPkg)

		mockPkg.On("CreateAgentPackage", ctx, mock.Anything, mock.AnythingOfType("string")).Return(nil, errMock)

		result, err := svc.CreateAgentPackage(ctx, apiPkg())

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "create agent package")
		mockPkg.AssertExpectations(t)
	})
}

func TestService_UpdateAgentPackage(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockPkg := new(mockAgentPackageUsecase)
		svc := newSvc(t, mockPkg)

		mockPkg.On("UpdateAgentPackage", ctx, "default", "pkg-1", mock.Anything).
			Return(newPkg(), nil)

		result, err := svc.UpdateAgentPackage(ctx, "default", "pkg-1", apiPkg())

		require.NoError(t, err)
		assert.Equal(t, "pkg-1", result.Metadata.Name)
		mockPkg.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockPkg := new(mockAgentPackageUsecase)
		svc := newSvc(t, mockPkg)

		mockPkg.On("UpdateAgentPackage", ctx, "default", "pkg-1", mock.Anything).Return(nil, errMock)

		result, err := svc.UpdateAgentPackage(ctx, "default", "pkg-1", apiPkg())

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "update agent package")
		mockPkg.AssertExpectations(t)
	})
}

func TestService_DeleteAgentPackage(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockPkg := new(mockAgentPackageUsecase)
		svc := newSvc(t, mockPkg)

		mockPkg.On("DeleteAgentPackage", ctx, "default", "pkg-1",
			mock.AnythingOfType("time.Time"), mock.AnythingOfType("string")).Return(nil)

		err := svc.DeleteAgentPackage(ctx, "default", "pkg-1")

		require.NoError(t, err)
		mockPkg.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockPkg := new(mockAgentPackageUsecase)
		svc := newSvc(t, mockPkg)

		mockPkg.On("DeleteAgentPackage", ctx, "default", "pkg-1",
			mock.AnythingOfType("time.Time"), mock.AnythingOfType("string")).Return(errMock)

		err := svc.DeleteAgentPackage(ctx, "default", "pkg-1")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "delete agent package")
		mockPkg.AssertExpectations(t)
	})
}
