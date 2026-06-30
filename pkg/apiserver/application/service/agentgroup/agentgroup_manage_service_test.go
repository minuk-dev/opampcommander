package agentgroup_test

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
	agentgroupsvc "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/agentgroup"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

var errMock = errors.New("mock error")

// mockAgentGroupUsecase is a mock implementation of agentport.AgentGroupUsecase.
type mockAgentGroupUsecase struct {
	mock.Mock
}

func (m *mockAgentGroupUsecase) GetAgentGroup(
	ctx context.Context, namespace, name string, options *model.GetOptions,
) (*agentmodel.AgentGroup, error) {
	args := m.Called(ctx, namespace, name, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	group, _ := args.Get(0).(*agentmodel.AgentGroup)

	return group, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockAgentGroupUsecase) ListAgentGroups(
	ctx context.Context, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.AgentGroup], error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, _ := args.Get(0).(*model.ListResponse[*agentmodel.AgentGroup])

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockAgentGroupUsecase) SaveAgentGroup(
	ctx context.Context, namespace, name string, agentGroup *agentmodel.AgentGroup,
) (*agentmodel.AgentGroup, error) {
	args := m.Called(ctx, namespace, name, agentGroup)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	group, _ := args.Get(0).(*agentmodel.AgentGroup)

	return group, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockAgentGroupUsecase) DeleteAgentGroup(
	ctx context.Context, namespace, name string, deletedAt time.Time, deletedBy string,
) error {
	args := m.Called(ctx, namespace, name, deletedAt, deletedBy)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func (m *mockAgentGroupUsecase) GetAgentGroupsForAgent(
	ctx context.Context, agent *agentmodel.Agent,
) ([]*agentmodel.AgentGroup, error) {
	args := m.Called(ctx, agent)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	groups, _ := args.Get(0).([]*agentmodel.AgentGroup)

	return groups, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockAgentGroupUsecase) PropagateAgentRemoteConfigChange(
	ctx context.Context, namespace, remoteConfigName string,
) error {
	args := m.Called(ctx, namespace, remoteConfigName)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func (m *mockAgentGroupUsecase) ApplyMatchingAgentGroupsToAgent(ctx context.Context, agent *agentmodel.Agent) error {
	args := m.Called(ctx, agent)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func (m *mockAgentGroupUsecase) ReconcileAgent(ctx context.Context, agent *agentmodel.Agent) error {
	args := m.Called(ctx, agent)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func (m *mockAgentGroupUsecase) ReconcileAgentGroup(ctx context.Context, namespace, name string) error {
	args := m.Called(ctx, namespace, name)

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

	agent, _ := args.Get(0).(*agentmodel.Agent)

	return agent, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockAgentUsecase) GetOrCreateAgent(ctx context.Context, instanceUID uuid.UUID) (*agentmodel.Agent, error) {
	args := m.Called(ctx, instanceUID)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	agent, _ := args.Get(0).(*agentmodel.Agent)

	return agent, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockAgentUsecase) ListAgentsBySelector(
	ctx context.Context, selector agentmodel.AgentSelector, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Agent], error) {
	args := m.Called(ctx, selector, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, _ := args.Get(0).(*model.ListResponse[*agentmodel.Agent])

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

	resp, _ := args.Get(0).(*model.ListResponse[*agentmodel.Agent])

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockAgentUsecase) SearchAgents(
	ctx context.Context, namespace string, query string, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Agent], error) {
	args := m.Called(ctx, namespace, query, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, _ := args.Get(0).(*model.ListResponse[*agentmodel.Agent])

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func newSvc(t *testing.T, group *mockAgentGroupUsecase, agent *mockAgentUsecase) *agentgroupsvc.ManageService {
	t.Helper()

	base := testutil.NewBase(t)

	return agentgroupsvc.NewManageService(group, agent, base.Logger)
}

func newGroup() *agentmodel.AgentGroup {
	return agentmodel.NewAgentGroup("default", "g-1", nil, time.Now(), "tester")
}

func apiGroup() *v1.AgentGroup {
	return &v1.AgentGroup{
		Kind:       v1.AgentGroupKind,
		APIVersion: v1.APIVersion,
		Metadata:   v1.Metadata{Namespace: "default", Name: "g-1"},
	}
}

func TestService_GetAgentGroup(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockGroup := new(mockAgentGroupUsecase)
		svc := newSvc(t, mockGroup, new(mockAgentUsecase))

		mockGroup.On("GetAgentGroup", ctx, "default", "g-1", (*model.GetOptions)(nil)).
			Return(newGroup(), nil)

		result, err := svc.GetAgentGroup(ctx, "default", "g-1", nil)

		require.NoError(t, err)
		assert.Equal(t, "g-1", result.Metadata.Name)
		mockGroup.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockGroup := new(mockAgentGroupUsecase)
		svc := newSvc(t, mockGroup, new(mockAgentUsecase))

		mockGroup.On("GetAgentGroup", ctx, "default", "missing", (*model.GetOptions)(nil)).Return(nil, errMock)

		result, err := svc.GetAgentGroup(ctx, "default", "missing", nil)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "get agent group")
		mockGroup.AssertExpectations(t)
	})
}

func TestService_ListAgentGroups(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockGroup := new(mockAgentGroupUsecase)
		svc := newSvc(t, mockGroup, new(mockAgentUsecase))

		opts := &applicationport.ListOptions{Limit: 10}
		resp := &model.ListResponse[*agentmodel.AgentGroup]{
			Items:    []*agentmodel.AgentGroup{newGroup()},
			Continue: "next",
		}
		mockGroup.On("ListAgentGroups", ctx, opts.ToDomain()).Return(resp, nil)

		result, err := svc.ListAgentGroups(ctx, opts)

		require.NoError(t, err)
		assert.Equal(t, v1.AgentGroupKind, result.Kind)
		assert.Len(t, result.Items, 1)
		assert.Equal(t, "next", result.Metadata.Continue)
		mockGroup.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockGroup := new(mockAgentGroupUsecase)
		svc := newSvc(t, mockGroup, new(mockAgentUsecase))

		opts := &applicationport.ListOptions{Limit: 10}
		mockGroup.On("ListAgentGroups", ctx, opts.ToDomain()).Return(nil, errMock)

		result, err := svc.ListAgentGroups(ctx, opts)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "list agent groups")
		mockGroup.AssertExpectations(t)
	})
}

func TestService_CreateAgentGroup(t *testing.T) {
	t.Parallel()

	t.Run("success when not already present", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockGroup := new(mockAgentGroupUsecase)
		svc := newSvc(t, mockGroup, new(mockAgentUsecase))

		mockGroup.On("GetAgentGroup", ctx, "default", "g-1", (*model.GetOptions)(nil)).
			Return(nil, model.ErrResourceNotExist)
		mockGroup.On("SaveAgentGroup", ctx, "default", "g-1", mock.Anything).
			Return(newGroup(), nil)

		result, err := svc.CreateAgentGroup(ctx, apiGroup())

		require.NoError(t, err)
		assert.Equal(t, "g-1", result.Metadata.Name)
		mockGroup.AssertExpectations(t)
	})

	t.Run("rejects when already exists", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockGroup := new(mockAgentGroupUsecase)
		svc := newSvc(t, mockGroup, new(mockAgentUsecase))

		mockGroup.On("GetAgentGroup", ctx, "default", "g-1", (*model.GetOptions)(nil)).
			Return(newGroup(), nil)

		result, err := svc.CreateAgentGroup(ctx, apiGroup())

		require.Error(t, err)
		assert.Nil(t, result)
		require.ErrorIs(t, err, agentgroupsvc.ErrAgentGroupAlreadyExists)
		mockGroup.AssertNotCalled(t, "SaveAgentGroup", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		mockGroup.AssertExpectations(t)
	})

	t.Run("save error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockGroup := new(mockAgentGroupUsecase)
		svc := newSvc(t, mockGroup, new(mockAgentUsecase))

		mockGroup.On("GetAgentGroup", ctx, "default", "g-1", (*model.GetOptions)(nil)).
			Return(nil, model.ErrResourceNotExist)
		mockGroup.On("SaveAgentGroup", ctx, "default", "g-1", mock.Anything).Return(nil, errMock)

		result, err := svc.CreateAgentGroup(ctx, apiGroup())

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "create agent group")
		mockGroup.AssertExpectations(t)
	})
}

func TestService_UpdateAgentGroup(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockGroup := new(mockAgentGroupUsecase)
		svc := newSvc(t, mockGroup, new(mockAgentUsecase))

		mockGroup.On("GetAgentGroup", ctx, "default", "g-1", (*model.GetOptions)(nil)).
			Return(newGroup(), nil)
		mockGroup.On("SaveAgentGroup", ctx, "default", "g-1", mock.Anything).
			Return(newGroup(), nil)

		result, err := svc.UpdateAgentGroup(ctx, "default", "g-1", apiGroup())

		require.NoError(t, err)
		assert.Equal(t, "g-1", result.Metadata.Name)
		mockGroup.AssertExpectations(t)
	})

	t.Run("get error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockGroup := new(mockAgentGroupUsecase)
		svc := newSvc(t, mockGroup, new(mockAgentUsecase))

		mockGroup.On("GetAgentGroup", ctx, "default", "g-1", (*model.GetOptions)(nil)).Return(nil, errMock)

		result, err := svc.UpdateAgentGroup(ctx, "default", "g-1", apiGroup())

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "get agent group for update")
		mockGroup.AssertExpectations(t)
	})
}

func TestService_DeleteAgentGroup(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockGroup := new(mockAgentGroupUsecase)
		svc := newSvc(t, mockGroup, new(mockAgentUsecase))

		mockGroup.On("DeleteAgentGroup", ctx, "default", "g-1",
			mock.AnythingOfType("time.Time"), mock.AnythingOfType("string")).Return(nil)

		err := svc.DeleteAgentGroup(ctx, "default", "g-1")

		require.NoError(t, err)
		mockGroup.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockGroup := new(mockAgentGroupUsecase)
		svc := newSvc(t, mockGroup, new(mockAgentUsecase))

		mockGroup.On("DeleteAgentGroup", ctx, "default", "g-1",
			mock.AnythingOfType("time.Time"), mock.AnythingOfType("string")).Return(errMock)

		err := svc.DeleteAgentGroup(ctx, "default", "g-1")

		require.Error(t, err)
		mockGroup.AssertExpectations(t)
	})
}

func TestService_ListAgentsByAgentGroup(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockGroup := new(mockAgentGroupUsecase)
		mockAgent := new(mockAgentUsecase)
		svc := newSvc(t, mockGroup, mockAgent)

		group := newGroup()
		mockGroup.On("GetAgentGroup", ctx, "default", "g-1", (*model.GetOptions)(nil)).Return(group, nil)
		mockAgent.On("ListAgentsBySelector", ctx, group.Spec.Selector, mock.Anything).
			Return(&model.ListResponse[*agentmodel.Agent]{Items: nil}, nil)

		result, err := svc.ListAgentsByAgentGroup(ctx, "default", "g-1", &applicationport.ListOptions{Limit: 10})

		require.NoError(t, err)
		assert.Equal(t, v1.AgentKind, result.Kind)
		mockGroup.AssertExpectations(t)
		mockAgent.AssertExpectations(t)
	})

	t.Run("group lookup error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockGroup := new(mockAgentGroupUsecase)
		svc := newSvc(t, mockGroup, new(mockAgentUsecase))

		mockGroup.On("GetAgentGroup", ctx, "default", "g-1", (*model.GetOptions)(nil)).Return(nil, errMock)

		result, err := svc.ListAgentsByAgentGroup(ctx, "default", "g-1", &applicationport.ListOptions{Limit: 10})

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "get agent group")
		mockGroup.AssertExpectations(t)
	})
}

func TestService_ListAgentGroupsByAgent(t *testing.T) {
	t.Parallel()

	t.Run("namespace mismatch returns ErrAgentNamespaceMismatch", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockGroup := new(mockAgentGroupUsecase)
		mockAgent := new(mockAgentUsecase)
		svc := newSvc(t, mockGroup, mockAgent)

		instanceUID := uuid.New()
		agent := agentmodel.NewAgent(instanceUID)
		agent.Metadata.Namespace = "other"
		mockAgent.On("GetAgent", ctx, instanceUID).Return(agent, nil)

		result, err := svc.ListAgentGroupsByAgent(ctx, "default", instanceUID)

		require.Error(t, err)
		assert.Nil(t, result)
		require.ErrorIs(t, err, applicationport.ErrAgentNamespaceMismatch)
		mockAgent.AssertExpectations(t)
	})

	t.Run("agent lookup error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockAgent := new(mockAgentUsecase)
		svc := newSvc(t, new(mockAgentGroupUsecase), mockAgent)

		instanceUID := uuid.New()
		mockAgent.On("GetAgent", ctx, instanceUID).Return(nil, errMock)

		result, err := svc.ListAgentGroupsByAgent(ctx, "default", instanceUID)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "get agent")
		mockAgent.AssertExpectations(t)
	})
}
