package endpoint_test

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
	endpointsvc "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/endpoint"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

var errMock = errors.New("mock error")

// mockEndpointUsecase is a mock implementation of agentport.EndpointUsecase.
type mockEndpointUsecase struct {
	mock.Mock
}

func (m *mockEndpointUsecase) GetEndpoint(
	ctx context.Context, namespace, name string, options *model.GetOptions,
) (*agentmodel.Endpoint, error) {
	args := m.Called(ctx, namespace, name, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	ep, ok := args.Get(0).(*agentmodel.Endpoint)
	if !ok {
		return nil, errMock
	}

	return ep, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockEndpointUsecase) ListEndpoints(
	ctx context.Context, namespace string, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Endpoint], error) {
	args := m.Called(ctx, namespace, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, ok := args.Get(0).(*model.ListResponse[*agentmodel.Endpoint])
	if !ok {
		return nil, errMock
	}

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockEndpointUsecase) SaveEndpoint(
	ctx context.Context, endpoint *agentmodel.Endpoint,
) (*agentmodel.Endpoint, error) {
	args := m.Called(ctx, endpoint)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	ep, ok := args.Get(0).(*agentmodel.Endpoint)
	if !ok {
		return nil, errMock
	}

	return ep, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockEndpointUsecase) CreateEndpoint(
	ctx context.Context, endpoint *agentmodel.Endpoint, actor string,
) (*agentmodel.Endpoint, error) {
	args := m.Called(ctx, endpoint, actor)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	ep, ok := args.Get(0).(*agentmodel.Endpoint)
	if !ok {
		return nil, errMock
	}

	return ep, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockEndpointUsecase) UpdateEndpoint(
	ctx context.Context, namespace, name string, endpoint *agentmodel.Endpoint,
) (*agentmodel.Endpoint, error) {
	args := m.Called(ctx, namespace, name, endpoint)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	ep, ok := args.Get(0).(*agentmodel.Endpoint)
	if !ok {
		return nil, errMock
	}

	return ep, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockEndpointUsecase) DeleteEndpoint(
	ctx context.Context, namespace, name string, deletedAt time.Time, deletedBy string,
) error {
	args := m.Called(ctx, namespace, name, deletedAt, deletedBy)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func newSvc(t *testing.T, ep *mockEndpointUsecase) *endpointsvc.Service {
	t.Helper()

	base := testutil.NewBase(t)

	return endpointsvc.NewEndpointService(ep, base.Logger)
}

func newEndpoint(namespace, name string) *agentmodel.Endpoint {
	return &agentmodel.Endpoint{
		Metadata: agentmodel.EndpointMetadata{Namespace: namespace, Name: name},
	}
}

func apiEndpoint(namespace, name string) *v1.Endpoint {
	return &v1.Endpoint{
		Kind:       v1.EndpointKind,
		APIVersion: v1.APIVersion,
		Metadata:   v1.EndpointMetadata{Namespace: namespace, Name: name},
	}
}

func TestService_GetEndpoint(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockEP := new(mockEndpointUsecase)
		svc := newSvc(t, mockEP)

		mockEP.On("GetEndpoint", ctx, "default", "ep-1", (*model.GetOptions)(nil)).
			Return(newEndpoint("default", "ep-1"), nil)

		result, err := svc.GetEndpoint(ctx, "default", "ep-1", nil)

		require.NoError(t, err)
		assert.Equal(t, "ep-1", result.Metadata.Name)
		mockEP.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockEP := new(mockEndpointUsecase)
		svc := newSvc(t, mockEP)

		mockEP.On("GetEndpoint", ctx, "default", "missing", (*model.GetOptions)(nil)).Return(nil, errMock)

		result, err := svc.GetEndpoint(ctx, "default", "missing", nil)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "get endpoint")
		mockEP.AssertExpectations(t)
	})
}

func TestService_ListEndpoints(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockEP := new(mockEndpointUsecase)
		svc := newSvc(t, mockEP)

		opts := &applicationport.ListOptions{Limit: 10}
		resp := &model.ListResponse[*agentmodel.Endpoint]{
			Items:    []*agentmodel.Endpoint{newEndpoint("default", "ep-1")},
			Continue: "next",
		}
		mockEP.On("ListEndpoints", ctx, "default", opts.ToDomain()).Return(resp, nil)

		result, err := svc.ListEndpoints(ctx, "default", opts)

		require.NoError(t, err)
		assert.Equal(t, v1.EndpointKind, result.Kind)
		assert.Len(t, result.Items, 1)
		assert.Equal(t, "next", result.Metadata.Continue)
		mockEP.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockEP := new(mockEndpointUsecase)
		svc := newSvc(t, mockEP)

		opts := &applicationport.ListOptions{Limit: 10}
		mockEP.On("ListEndpoints", ctx, "default", opts.ToDomain()).Return(nil, errMock)

		result, err := svc.ListEndpoints(ctx, "default", opts)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "list endpoints")
		mockEP.AssertExpectations(t)
	})
}

func TestService_CreateEndpoint(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockEP := new(mockEndpointUsecase)
		svc := newSvc(t, mockEP)

		mockEP.On("CreateEndpoint", ctx, mock.MatchedBy(func(e *agentmodel.Endpoint) bool {
			return e.Metadata.Name == "ep-1"
		}), mock.AnythingOfType("string")).Return(newEndpoint("default", "ep-1"), nil)

		result, err := svc.CreateEndpoint(ctx, apiEndpoint("default", "ep-1"))

		require.NoError(t, err)
		assert.Equal(t, "ep-1", result.Metadata.Name)
		mockEP.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockEP := new(mockEndpointUsecase)
		svc := newSvc(t, mockEP)

		mockEP.On("CreateEndpoint", ctx, mock.Anything, mock.AnythingOfType("string")).Return(nil, errMock)

		result, err := svc.CreateEndpoint(ctx, apiEndpoint("default", "ep-1"))

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "create endpoint")
		mockEP.AssertExpectations(t)
	})
}

func TestService_UpdateEndpoint(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockEP := new(mockEndpointUsecase)
		svc := newSvc(t, mockEP)

		mockEP.On("UpdateEndpoint", ctx, "default", "ep-1", mock.Anything).
			Return(newEndpoint("default", "ep-1"), nil)

		result, err := svc.UpdateEndpoint(ctx, "default", "ep-1", apiEndpoint("default", "ep-1"))

		require.NoError(t, err)
		assert.Equal(t, "ep-1", result.Metadata.Name)
		mockEP.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockEP := new(mockEndpointUsecase)
		svc := newSvc(t, mockEP)

		mockEP.On("UpdateEndpoint", ctx, "default", "ep-1", mock.Anything).Return(nil, errMock)

		result, err := svc.UpdateEndpoint(ctx, "default", "ep-1", apiEndpoint("default", "ep-1"))

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "update endpoint")
		mockEP.AssertExpectations(t)
	})
}

func TestService_DeleteEndpoint(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockEP := new(mockEndpointUsecase)
		svc := newSvc(t, mockEP)

		mockEP.On("DeleteEndpoint", ctx, "default", "ep-1",
			mock.AnythingOfType("time.Time"), mock.AnythingOfType("string")).Return(nil)

		err := svc.DeleteEndpoint(ctx, "default", "ep-1")

		require.NoError(t, err)
		mockEP.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockEP := new(mockEndpointUsecase)
		svc := newSvc(t, mockEP)

		mockEP.On("DeleteEndpoint", ctx, "default", "ep-1",
			mock.AnythingOfType("time.Time"), mock.AnythingOfType("string")).Return(errMock)

		err := svc.DeleteEndpoint(ctx, "default", "ep-1")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "delete endpoint")
		mockEP.AssertExpectations(t)
	})
}
