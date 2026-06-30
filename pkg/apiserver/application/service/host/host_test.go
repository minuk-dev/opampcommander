package host_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	applicationport "github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	hostsvc "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/host"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

var errMock = errors.New("mock error")

// mockHostUsecase is a mock implementation of agentport.HostUsecase.
type mockHostUsecase struct {
	mock.Mock
}

func (m *mockHostUsecase) GetHost(ctx context.Context, id string) (*agentmodel.Host, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	host, ok := args.Get(0).(*agentmodel.Host)
	if !ok {
		return nil, errMock
	}

	return host, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockHostUsecase) ListHosts(
	ctx context.Context, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Host], error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, ok := args.Get(0).(*model.ListResponse[*agentmodel.Host])
	if !ok {
		return nil, errMock
	}

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockHostUsecase) ObserveAgent(ctx context.Context, agent *agentmodel.Agent) error {
	args := m.Called(ctx, agent)

	return args.Error(0) //nolint:wrapcheck // mock error
}

// mockAgentUsecase is a mock implementation of agentport.AgentUsecase.
type mockAgentUsecase struct {
	mock.Mock
}

func (m *mockAgentUsecase) GetAgent(ctx context.Context, instanceUID uuid.UUID) (*agentmodel.Agent, error) {
	args := m.Called(ctx, instanceUID)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	agent, ok := args.Get(0).(*agentmodel.Agent)
	if !ok {
		return nil, errMock
	}

	return agent, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockAgentUsecase) GetOrCreateAgent(ctx context.Context, instanceUID uuid.UUID) (*agentmodel.Agent, error) {
	args := m.Called(ctx, instanceUID)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	agent, ok := args.Get(0).(*agentmodel.Agent)
	if !ok {
		return nil, errMock
	}

	return agent, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockAgentUsecase) ListAgentsBySelector(
	ctx context.Context, selector agentmodel.AgentSelector, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Agent], error) {
	args := m.Called(ctx, selector, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, ok := args.Get(0).(*model.ListResponse[*agentmodel.Agent])
	if !ok {
		return nil, errMock
	}

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockAgentUsecase) SaveAgent(ctx context.Context, agent *agentmodel.Agent) error {
	args := m.Called(ctx, agent)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func (m *mockAgentUsecase) DeleteAgent(ctx context.Context, instanceUID uuid.UUID) error {
	args := m.Called(ctx, instanceUID)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func (m *mockAgentUsecase) ListAgents(
	ctx context.Context, namespace string, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Agent], error) {
	args := m.Called(ctx, namespace, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, ok := args.Get(0).(*model.ListResponse[*agentmodel.Agent])
	if !ok {
		return nil, errMock
	}

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockAgentUsecase) SearchAgents(
	ctx context.Context, namespace string, query string, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Agent], error) {
	args := m.Called(ctx, namespace, query, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, ok := args.Get(0).(*model.ListResponse[*agentmodel.Agent])
	if !ok {
		return nil, errMock
	}

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func newSvc(t *testing.T, host *mockHostUsecase, agent *mockAgentUsecase) *hostsvc.Service {
	t.Helper()

	base := testutil.NewBase(t)

	return hostsvc.New(host, agent, base.Logger)
}

func newHost(id string) *agentmodel.Host {
	return &agentmodel.Host{
		Metadata: agentmodel.HostMetadata{
			ID:          id,
			Name:        "host-" + id,
			FirstSeenAt: time.Now(),
			LastSeenAt:  time.Now(),
		},
		Status: agentmodel.HostStatus{},
	}
}

func TestService_GetHost(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockHost := new(mockHostUsecase)
		svc := newSvc(t, mockHost, new(mockAgentUsecase))

		mockHost.On("GetHost", ctx, "host-1").Return(newHost("host-1"), nil)

		result, err := svc.GetHost(ctx, "host-1")

		require.NoError(t, err)
		assert.Equal(t, v1.HostKind, result.Kind)
		assert.Equal(t, "host-1", result.Metadata.ID)
		mockHost.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockHost := new(mockHostUsecase)
		svc := newSvc(t, mockHost, new(mockAgentUsecase))

		mockHost.On("GetHost", ctx, "missing").Return(nil, errMock)

		result, err := svc.GetHost(ctx, "missing")

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to get host")
		mockHost.AssertExpectations(t)
	})
}

func TestService_ListHosts(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockHost := new(mockHostUsecase)
		svc := newSvc(t, mockHost, new(mockAgentUsecase))

		opts := &applicationport.ListOptions{Limit: 10}
		resp := &model.ListResponse[*agentmodel.Host]{
			Items:    []*agentmodel.Host{newHost("host-1"), newHost("host-2")},
			Continue: "next",
		}
		mockHost.On("ListHosts", ctx, opts.ToDomain()).Return(resp, nil)

		result, err := svc.ListHosts(ctx, opts)

		require.NoError(t, err)
		assert.Equal(t, v1.HostKind, result.Kind)
		assert.Len(t, result.Items, 2)
		assert.Equal(t, "next", result.Metadata.Continue)
		mockHost.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockHost := new(mockHostUsecase)
		svc := newSvc(t, mockHost, new(mockAgentUsecase))

		opts := &applicationport.ListOptions{Limit: 10}
		mockHost.On("ListHosts", ctx, opts.ToDomain()).Return(nil, errMock)

		result, err := svc.ListHosts(ctx, opts)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to list hosts")
		mockHost.AssertExpectations(t)
	})
}

func TestService_ListAgentsByHost(t *testing.T) {
	t.Parallel()

	t.Run("host with no agents returns empty list", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockHost := new(mockHostUsecase)
		mockAgent := new(mockAgentUsecase)
		svc := newSvc(t, mockHost, mockAgent)

		mockHost.On("GetHost", ctx, "host-1").Return(newHost("host-1"), nil)

		result, err := svc.ListAgentsByHost(ctx, "host-1", &applicationport.ListOptions{Limit: 10})

		require.NoError(t, err)
		assert.Equal(t, v1.AgentKind, result.Kind)
		assert.Empty(t, result.Items)
		// No agent lookups should happen when the host has no associated agents.
		mockAgent.AssertNotCalled(t, "GetAgent", mock.Anything, mock.Anything)
		mockHost.AssertExpectations(t)
	})

	t.Run("host lookup error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockHost := new(mockHostUsecase)
		svc := newSvc(t, mockHost, new(mockAgentUsecase))

		mockHost.On("GetHost", ctx, "missing").Return(nil, errMock)

		result, err := svc.ListAgentsByHost(ctx, "missing", &applicationport.ListOptions{Limit: 10})

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to get host")
		mockHost.AssertExpectations(t)
	})
}
