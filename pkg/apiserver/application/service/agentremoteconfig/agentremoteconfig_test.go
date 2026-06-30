package agentremoteconfig_test

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
	arcsvc "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/agentremoteconfig"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

var errMock = errors.New("mock error")

// mockAgentRemoteConfigUsecase is a mock implementation of agentport.AgentRemoteConfigUsecase.
type mockAgentRemoteConfigUsecase struct {
	mock.Mock
}

func (m *mockAgentRemoteConfigUsecase) GetAgentRemoteConfig(
	ctx context.Context, namespace, name string, options *model.GetOptions,
) (*agentmodel.AgentRemoteConfig, error) {
	args := m.Called(ctx, namespace, name, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	cfg, ok := args.Get(0).(*agentmodel.AgentRemoteConfig)
	if !ok {
		return nil, errMock
	}

	return cfg, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockAgentRemoteConfigUsecase) ListAgentRemoteConfigs(
	ctx context.Context, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.AgentRemoteConfig], error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, ok := args.Get(0).(*model.ListResponse[*agentmodel.AgentRemoteConfig])
	if !ok {
		return nil, errMock
	}

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockAgentRemoteConfigUsecase) SaveAgentRemoteConfig(
	ctx context.Context, agentRemoteConfig *agentmodel.AgentRemoteConfig,
) (*agentmodel.AgentRemoteConfig, error) {
	args := m.Called(ctx, agentRemoteConfig)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	cfg, ok := args.Get(0).(*agentmodel.AgentRemoteConfig)
	if !ok {
		return nil, errMock
	}

	return cfg, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockAgentRemoteConfigUsecase) CreateAgentRemoteConfig(
	ctx context.Context, agentRemoteConfig *agentmodel.AgentRemoteConfig, actor string,
) (*agentmodel.AgentRemoteConfig, error) {
	args := m.Called(ctx, agentRemoteConfig, actor)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	cfg, ok := args.Get(0).(*agentmodel.AgentRemoteConfig)
	if !ok {
		return nil, errMock
	}

	return cfg, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockAgentRemoteConfigUsecase) UpdateAgentRemoteConfig(
	ctx context.Context, namespace, name string, agentRemoteConfig *agentmodel.AgentRemoteConfig,
) (*agentmodel.AgentRemoteConfig, error) {
	args := m.Called(ctx, namespace, name, agentRemoteConfig)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	cfg, ok := args.Get(0).(*agentmodel.AgentRemoteConfig)
	if !ok {
		return nil, errMock
	}

	return cfg, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockAgentRemoteConfigUsecase) DeleteAgentRemoteConfig(
	ctx context.Context, namespace, name string, deletedAt time.Time, deletedBy string,
) error {
	args := m.Called(ctx, namespace, name, deletedAt, deletedBy)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func (m *mockAgentRemoteConfigUsecase) ReconcileAgentRemoteConfig(
	ctx context.Context, namespace, name string,
) error {
	args := m.Called(ctx, namespace, name)

	return args.Error(0) //nolint:wrapcheck // mock error
}

// stubAgentGroupUsecase is a no-op agentport.AgentGroupUsecase. PropagateAgentRemoteConfigChange
// signals propagateCh so a test can wait for the fire-and-forget propagation goroutine to run
// before completing (avoiding goroutine leaks racing the test).
type stubAgentGroupUsecase struct {
	propagateCh chan struct{}
}

func (s *stubAgentGroupUsecase) PropagateAgentRemoteConfigChange(_ context.Context, _, _ string) error {
	if s.propagateCh != nil {
		s.propagateCh <- struct{}{}
	}

	return nil
}

func (*stubAgentGroupUsecase) GetAgentGroup(
	context.Context, string, string, *model.GetOptions,
) (*agentmodel.AgentGroup, error) {
	return nil, nil //nolint:nilnil // stub
}

func (*stubAgentGroupUsecase) ListAgentGroups(
	context.Context, *model.ListOptions,
) (*model.ListResponse[*agentmodel.AgentGroup], error) {
	return nil, nil //nolint:nilnil // stub
}

func (*stubAgentGroupUsecase) SaveAgentGroup(
	context.Context, string, string, *agentmodel.AgentGroup,
) (*agentmodel.AgentGroup, error) {
	return nil, nil //nolint:nilnil // stub
}

func (*stubAgentGroupUsecase) DeleteAgentGroup(context.Context, string, string, time.Time, string) error {
	return nil
}

func (*stubAgentGroupUsecase) GetAgentGroupsForAgent(
	context.Context, *agentmodel.Agent,
) ([]*agentmodel.AgentGroup, error) {
	return nil, nil
}

func (*stubAgentGroupUsecase) ApplyMatchingAgentGroupsToAgent(context.Context, *agentmodel.Agent) error {
	return nil
}

func (*stubAgentGroupUsecase) ReconcileAgent(context.Context, *agentmodel.Agent) error { return nil }

func (*stubAgentGroupUsecase) ReconcileAgentGroup(context.Context, string, string) error { return nil }

// stubEndpointDetectionUsecase is a no-op agentport.EndpointDetectionUsecase.
// ReconcileEndpointsFromRemoteConfig signals detectCh so a test can wait for the
// fire-and-forget detection goroutine to run.
type stubEndpointDetectionUsecase struct {
	detectCh chan struct{}
}

func (s *stubEndpointDetectionUsecase) ReconcileEndpointsFromRemoteConfig(
	context.Context, *agentmodel.AgentRemoteConfig,
) error {
	if s.detectCh != nil {
		s.detectCh <- struct{}{}
	}

	return nil
}

func (*stubEndpointDetectionUsecase) ExtractEndpointsFromAgent(
	*agentmodel.Agent,
) ([]*agentmodel.Endpoint, error) {
	return nil, nil
}

func newSvc(
	t *testing.T, arc *mockAgentRemoteConfigUsecase, group *stubAgentGroupUsecase, det *stubEndpointDetectionUsecase,
) *arcsvc.Service {
	t.Helper()

	base := testutil.NewBase(t)

	return arcsvc.NewAgentRemoteConfigService(arc, group, det, base.Logger)
}

func newARC(namespace, name string) *agentmodel.AgentRemoteConfig {
	return &agentmodel.AgentRemoteConfig{
		Metadata: agentmodel.AgentRemoteConfigMetadata{Namespace: namespace, Name: name},
	}
}

func apiARC(namespace, name string) *v1.AgentRemoteConfig {
	return &v1.AgentRemoteConfig{
		Kind:       v1.AgentRemoteConfigKind,
		APIVersion: v1.APIVersion,
		Metadata:   v1.AgentRemoteConfigMetadata{Namespace: namespace, Name: name},
	}
}

// waitSignal fails the test if the fire-and-forget goroutine does not signal in time.
func waitSignal(t *testing.T, ch chan struct{}) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for background side effect")
	}
}

func TestService_GetAgentRemoteConfig(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockARC := new(mockAgentRemoteConfigUsecase)
		svc := newSvc(t, mockARC, &stubAgentGroupUsecase{}, &stubEndpointDetectionUsecase{})

		mockARC.On("GetAgentRemoteConfig", ctx, "default", "cfg-1", (*model.GetOptions)(nil)).
			Return(newARC("default", "cfg-1"), nil)

		result, err := svc.GetAgentRemoteConfig(ctx, "default", "cfg-1", nil)

		require.NoError(t, err)
		assert.Equal(t, "cfg-1", result.Metadata.Name)
		mockARC.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockARC := new(mockAgentRemoteConfigUsecase)
		svc := newSvc(t, mockARC, &stubAgentGroupUsecase{}, &stubEndpointDetectionUsecase{})

		mockARC.On("GetAgentRemoteConfig", ctx, "default", "missing", (*model.GetOptions)(nil)).Return(nil, errMock)

		result, err := svc.GetAgentRemoteConfig(ctx, "default", "missing", nil)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "get agent remote config")
		mockARC.AssertExpectations(t)
	})
}

func TestService_ListAgentRemoteConfigs(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockARC := new(mockAgentRemoteConfigUsecase)
		svc := newSvc(t, mockARC, &stubAgentGroupUsecase{}, &stubEndpointDetectionUsecase{})

		opts := &applicationport.ListOptions{Limit: 10}
		resp := &model.ListResponse[*agentmodel.AgentRemoteConfig]{
			Items:    []*agentmodel.AgentRemoteConfig{newARC("default", "cfg-1")},
			Continue: "next",
		}
		mockARC.On("ListAgentRemoteConfigs", ctx, opts.ToDomain()).Return(resp, nil)

		result, err := svc.ListAgentRemoteConfigs(ctx, opts)

		require.NoError(t, err)
		assert.Equal(t, v1.AgentRemoteConfigKind, result.Kind)
		assert.Len(t, result.Items, 1)
		assert.Equal(t, "next", result.Metadata.Continue)
		mockARC.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockARC := new(mockAgentRemoteConfigUsecase)
		svc := newSvc(t, mockARC, &stubAgentGroupUsecase{}, &stubEndpointDetectionUsecase{})

		opts := &applicationport.ListOptions{Limit: 10}
		mockARC.On("ListAgentRemoteConfigs", ctx, opts.ToDomain()).Return(nil, errMock)

		result, err := svc.ListAgentRemoteConfigs(ctx, opts)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "list agent remote configs")
		mockARC.AssertExpectations(t)
	})
}

func TestService_CreateAgentRemoteConfig(t *testing.T) {
	t.Parallel()

	t.Run("success triggers propagation and endpoint detection", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockARC := new(mockAgentRemoteConfigUsecase)
		group := &stubAgentGroupUsecase{propagateCh: make(chan struct{}, 1)}
		det := &stubEndpointDetectionUsecase{detectCh: make(chan struct{}, 1)}
		svc := newSvc(t, mockARC, group, det)

		mockARC.On("CreateAgentRemoteConfig", ctx, mock.Anything, mock.AnythingOfType("string")).
			Return(newARC("default", "cfg-1"), nil)

		result, err := svc.CreateAgentRemoteConfig(ctx, apiARC("default", "cfg-1"))

		require.NoError(t, err)
		assert.Equal(t, "cfg-1", result.Metadata.Name)
		waitSignal(t, group.propagateCh)
		waitSignal(t, det.detectCh)
		mockARC.AssertExpectations(t)
	})

	t.Run("error skips side effects", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockARC := new(mockAgentRemoteConfigUsecase)
		svc := newSvc(t, mockARC, &stubAgentGroupUsecase{}, &stubEndpointDetectionUsecase{})

		mockARC.On("CreateAgentRemoteConfig", ctx, mock.Anything, mock.AnythingOfType("string")).Return(nil, errMock)

		result, err := svc.CreateAgentRemoteConfig(ctx, apiARC("default", "cfg-1"))

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "create agent remote config")
		mockARC.AssertExpectations(t)
	})
}

func TestService_UpdateAgentRemoteConfig(t *testing.T) {
	t.Parallel()

	t.Run("success triggers propagation and endpoint detection", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockARC := new(mockAgentRemoteConfigUsecase)
		group := &stubAgentGroupUsecase{propagateCh: make(chan struct{}, 1)}
		det := &stubEndpointDetectionUsecase{detectCh: make(chan struct{}, 1)}
		svc := newSvc(t, mockARC, group, det)

		mockARC.On("UpdateAgentRemoteConfig", ctx, "default", "cfg-1", mock.Anything).
			Return(newARC("default", "cfg-1"), nil)

		result, err := svc.UpdateAgentRemoteConfig(ctx, "default", "cfg-1", apiARC("default", "cfg-1"))

		require.NoError(t, err)
		assert.Equal(t, "cfg-1", result.Metadata.Name)
		waitSignal(t, group.propagateCh)
		waitSignal(t, det.detectCh)
		mockARC.AssertExpectations(t)
	})

	t.Run("error skips side effects", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockARC := new(mockAgentRemoteConfigUsecase)
		svc := newSvc(t, mockARC, &stubAgentGroupUsecase{}, &stubEndpointDetectionUsecase{})

		mockARC.On("UpdateAgentRemoteConfig", ctx, "default", "cfg-1", mock.Anything).Return(nil, errMock)

		result, err := svc.UpdateAgentRemoteConfig(ctx, "default", "cfg-1", apiARC("default", "cfg-1"))

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "update agent remote config")
		mockARC.AssertExpectations(t)
	})
}

func TestService_DeleteAgentRemoteConfig(t *testing.T) {
	t.Parallel()

	t.Run("success triggers propagation", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockARC := new(mockAgentRemoteConfigUsecase)
		group := &stubAgentGroupUsecase{propagateCh: make(chan struct{}, 1)}
		svc := newSvc(t, mockARC, group, &stubEndpointDetectionUsecase{})

		mockARC.On("DeleteAgentRemoteConfig", ctx, "default", "cfg-1",
			mock.AnythingOfType("time.Time"), mock.AnythingOfType("string")).Return(nil)

		err := svc.DeleteAgentRemoteConfig(ctx, "default", "cfg-1")

		require.NoError(t, err)
		waitSignal(t, group.propagateCh)
		mockARC.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockARC := new(mockAgentRemoteConfigUsecase)
		svc := newSvc(t, mockARC, &stubAgentGroupUsecase{}, &stubEndpointDetectionUsecase{})

		mockARC.On("DeleteAgentRemoteConfig", ctx, "default", "cfg-1",
			mock.AnythingOfType("time.Time"), mock.AnythingOfType("string")).Return(errMock)

		err := svc.DeleteAgentRemoteConfig(ctx, "default", "cfg-1")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "delete agent remote config")
		mockARC.AssertExpectations(t)
	})
}
