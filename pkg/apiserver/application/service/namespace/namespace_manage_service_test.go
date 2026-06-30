package namespace_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	applicationport "github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	namespacesvc "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/namespace"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

var errMock = errors.New("mock error")

// mockNamespaceUsecase is a mock implementation of agentport.NamespaceUsecase.
type mockNamespaceUsecase struct {
	mock.Mock
}

func (m *mockNamespaceUsecase) GetNamespace(
	ctx context.Context, name string, options *model.GetOptions,
) (*agentmodel.Namespace, error) {
	args := m.Called(ctx, name, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	ns, ok := args.Get(0).(*agentmodel.Namespace)
	if !ok {
		return nil, errMock
	}

	return ns, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockNamespaceUsecase) ListNamespaces(
	ctx context.Context, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Namespace], error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, ok := args.Get(0).(*model.ListResponse[*agentmodel.Namespace])
	if !ok {
		return nil, errMock
	}

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockNamespaceUsecase) SaveNamespace(
	ctx context.Context, namespace *agentmodel.Namespace,
) (*agentmodel.Namespace, error) {
	args := m.Called(ctx, namespace)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	ns, ok := args.Get(0).(*agentmodel.Namespace)
	if !ok {
		return nil, errMock
	}

	return ns, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockNamespaceUsecase) CreateNamespace(
	ctx context.Context, namespace *agentmodel.Namespace, actor string,
) (*agentmodel.Namespace, error) {
	args := m.Called(ctx, namespace, actor)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	ns, ok := args.Get(0).(*agentmodel.Namespace)
	if !ok {
		return nil, errMock
	}

	return ns, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockNamespaceUsecase) UpdateNamespace(
	ctx context.Context, name string, namespace *agentmodel.Namespace,
) (*agentmodel.Namespace, error) {
	args := m.Called(ctx, name, namespace)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	ns, ok := args.Get(0).(*agentmodel.Namespace)
	if !ok {
		return nil, errMock
	}

	return ns, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockNamespaceUsecase) DeleteNamespace(ctx context.Context, name string, actor string) error {
	args := m.Called(ctx, name, actor)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func newSvc(t *testing.T, ns *mockNamespaceUsecase) *namespacesvc.Service {
	t.Helper()

	base := testutil.NewBase(t)

	return namespacesvc.NewNamespaceService(ns, base.Logger)
}

func apiNamespace() *v1.Namespace {
	return &v1.Namespace{
		Kind:       v1.NamespaceKind,
		APIVersion: v1.APIVersion,
		Metadata:   v1.NamespaceMetadata{Name: "production"},
	}
}

func TestService_GetNamespace(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockNS := new(mockNamespaceUsecase)
		svc := newSvc(t, mockNS)

		mockNS.On("GetNamespace", ctx, "production", (*model.GetOptions)(nil)).
			Return(agentmodel.NewNamespace("production"), nil)

		result, err := svc.GetNamespace(ctx, "production", nil)

		require.NoError(t, err)
		assert.Equal(t, v1.NamespaceKind, result.Kind)
		assert.Equal(t, "production", result.Metadata.Name)
		mockNS.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockNS := new(mockNamespaceUsecase)
		svc := newSvc(t, mockNS)

		mockNS.On("GetNamespace", ctx, "missing", (*model.GetOptions)(nil)).Return(nil, errMock)

		result, err := svc.GetNamespace(ctx, "missing", nil)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "get namespace")
		mockNS.AssertExpectations(t)
	})
}

func TestService_ListNamespaces(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockNS := new(mockNamespaceUsecase)
		svc := newSvc(t, mockNS)

		opts := &applicationport.ListOptions{Limit: 10}
		resp := &model.ListResponse[*agentmodel.Namespace]{
			Items:    []*agentmodel.Namespace{agentmodel.NewNamespace("default"), agentmodel.NewNamespace("prod")},
			Continue: "next",
		}
		mockNS.On("ListNamespaces", ctx, opts.ToDomain()).Return(resp, nil)

		result, err := svc.ListNamespaces(ctx, opts)

		require.NoError(t, err)
		assert.Equal(t, v1.NamespaceKind, result.Kind)
		assert.Len(t, result.Items, 2)
		assert.Equal(t, "next", result.Metadata.Continue)
		mockNS.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockNS := new(mockNamespaceUsecase)
		svc := newSvc(t, mockNS)

		opts := &applicationport.ListOptions{Limit: 10}
		mockNS.On("ListNamespaces", ctx, opts.ToDomain()).Return(nil, errMock)

		result, err := svc.ListNamespaces(ctx, opts)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "list namespaces")
		mockNS.AssertExpectations(t)
	})
}

func TestService_CreateNamespace(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockNS := new(mockNamespaceUsecase)
		svc := newSvc(t, mockNS)

		// actor resolves to the anonymous user when no user is present in the context.
		mockNS.On("CreateNamespace", ctx, mock.MatchedBy(func(ns *agentmodel.Namespace) bool {
			return ns.Metadata.Name == "production"
		}), mock.AnythingOfType("string")).Return(agentmodel.NewNamespace("production"), nil)

		result, err := svc.CreateNamespace(ctx, apiNamespace())

		require.NoError(t, err)
		assert.Equal(t, "production", result.Metadata.Name)
		mockNS.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockNS := new(mockNamespaceUsecase)
		svc := newSvc(t, mockNS)

		mockNS.On("CreateNamespace", ctx, mock.Anything, mock.AnythingOfType("string")).
			Return(nil, errMock)

		result, err := svc.CreateNamespace(ctx, apiNamespace())

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "create namespace")
		mockNS.AssertExpectations(t)
	})
}

func TestService_UpdateNamespace(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockNS := new(mockNamespaceUsecase)
		svc := newSvc(t, mockNS)

		mockNS.On("UpdateNamespace", ctx, "production", mock.Anything).
			Return(agentmodel.NewNamespace("production"), nil)

		result, err := svc.UpdateNamespace(ctx, "production", apiNamespace())

		require.NoError(t, err)
		assert.Equal(t, "production", result.Metadata.Name)
		mockNS.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockNS := new(mockNamespaceUsecase)
		svc := newSvc(t, mockNS)

		mockNS.On("UpdateNamespace", ctx, "production", mock.Anything).Return(nil, errMock)

		result, err := svc.UpdateNamespace(ctx, "production", apiNamespace())

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "update namespace")
		mockNS.AssertExpectations(t)
	})
}

func TestService_DeleteNamespace(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockNS := new(mockNamespaceUsecase)
		svc := newSvc(t, mockNS)

		mockNS.On("DeleteNamespace", ctx, "production", mock.AnythingOfType("string")).Return(nil)

		err := svc.DeleteNamespace(ctx, "production")

		require.NoError(t, err)
		mockNS.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockNS := new(mockNamespaceUsecase)
		svc := newSvc(t, mockNS)

		mockNS.On("DeleteNamespace", ctx, "production", mock.AnythingOfType("string")).Return(errMock)

		err := svc.DeleteNamespace(ctx, "production")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "delete namespace")
		mockNS.AssertExpectations(t)
	})
}
