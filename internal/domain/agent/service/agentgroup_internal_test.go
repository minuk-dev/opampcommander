package agentservice

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/agent/model"
	"github.com/minuk-dev/opampcommander/internal/domain/agent/model/agent"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

var errRemoteConfigNotFound = errors.New("remote config not found")

// mockAgentGroupPersistence is a mock for AgentGroupPersistencePort.
type mockAgentGroupPersistence struct {
	mock.Mock
}

func (m *mockAgentGroupPersistence) GetAgentGroup(
	ctx context.Context,
	namespace string,
	name string,
	options *model.GetOptions,
) (*agentmodel.AgentGroup, error) {
	args := m.Called(ctx, namespace, name, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck
	}

	result, ok := args.Get(0).(*agentmodel.AgentGroup)
	if !ok {
		return nil, errUnexpectedType
	}

	return result, args.Error(1) //nolint:wrapcheck
}

func (m *mockAgentGroupPersistence) PutAgentGroup(
	ctx context.Context,
	namespace string,
	name string,
	ag *agentmodel.AgentGroup,
) (*agentmodel.AgentGroup, error) {
	args := m.Called(ctx, namespace, name, ag)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck
	}

	result, ok := args.Get(0).(*agentmodel.AgentGroup)
	if !ok {
		return nil, errUnexpectedType
	}

	return result, args.Error(1) //nolint:wrapcheck
}

func (m *mockAgentGroupPersistence) ListAgentGroups(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.AgentGroup], error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck
	}

	result, ok := args.Get(0).(*model.ListResponse[*agentmodel.AgentGroup])
	if !ok {
		return nil, errUnexpectedType
	}

	return result, args.Error(1) //nolint:wrapcheck
}

// mockAgentUsecase is a mock for AgentUsecase.
type mockAgentUsecase struct {
	mock.Mock
}

func (m *mockAgentUsecase) GetAgent(ctx context.Context, uid uuid.UUID) (*agentmodel.Agent, error) {
	args := m.Called(ctx, uid)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck
	}

	result, ok := args.Get(0).(*agentmodel.Agent)
	if !ok {
		return nil, errUnexpectedType
	}

	return result, args.Error(1) //nolint:wrapcheck
}

func (m *mockAgentUsecase) GetOrCreateAgent(ctx context.Context, uid uuid.UUID) (*agentmodel.Agent, error) {
	args := m.Called(ctx, uid)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck
	}

	result, ok := args.Get(0).(*agentmodel.Agent)
	if !ok {
		return nil, errUnexpectedType
	}

	return result, args.Error(1) //nolint:wrapcheck
}

func (m *mockAgentUsecase) ListAgentsBySelector(
	ctx context.Context,
	selector agentmodel.AgentSelector,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Agent], error) {
	args := m.Called(ctx, selector, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck
	}

	result, ok := args.Get(0).(*model.ListResponse[*agentmodel.Agent])
	if !ok {
		return nil, errUnexpectedType
	}

	return result, args.Error(1) //nolint:wrapcheck
}

func (m *mockAgentUsecase) SaveAgent(ctx context.Context, a *agentmodel.Agent) error {
	args := m.Called(ctx, a)

	return args.Error(0) //nolint:wrapcheck
}

func (m *mockAgentUsecase) DeleteAgent(ctx context.Context, instanceUID uuid.UUID) error {
	args := m.Called(ctx, instanceUID)

	return args.Error(0) //nolint:wrapcheck
}

func (m *mockAgentUsecase) ListAgents(
	ctx context.Context,
	namespace string,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Agent], error) {
	args := m.Called(ctx, namespace, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck
	}

	result, ok := args.Get(0).(*model.ListResponse[*agentmodel.Agent])
	if !ok {
		return nil, errUnexpectedType
	}

	return result, args.Error(1) //nolint:wrapcheck
}

func (m *mockAgentUsecase) SearchAgents(
	ctx context.Context,
	namespace string,
	query string,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Agent], error) {
	args := m.Called(ctx, namespace, query, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck
	}

	result, ok := args.Get(0).(*model.ListResponse[*agentmodel.Agent])
	if !ok {
		return nil, errUnexpectedType
	}

	return result, args.Error(1) //nolint:wrapcheck
}

// mockRemoteConfigPersistence is a mock for AgentRemoteConfigPersistencePort.
type mockRemoteConfigPersistence struct {
	mock.Mock
}

func (m *mockRemoteConfigPersistence) GetAgentRemoteConfig(
	ctx context.Context,
	namespace string,
	name string,
	options *model.GetOptions,
) (*agentmodel.AgentRemoteConfig, error) {
	args := m.Called(ctx, namespace, name, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck
	}

	result, ok := args.Get(0).(*agentmodel.AgentRemoteConfig)
	if !ok {
		return nil, errUnexpectedType
	}

	return result, args.Error(1) //nolint:wrapcheck
}

func (m *mockRemoteConfigPersistence) PutAgentRemoteConfig(
	ctx context.Context,
	config *agentmodel.AgentRemoteConfig,
) (*agentmodel.AgentRemoteConfig, error) {
	args := m.Called(ctx, config)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck
	}

	result, ok := args.Get(0).(*agentmodel.AgentRemoteConfig)
	if !ok {
		return nil, errUnexpectedType
	}

	return result, args.Error(1) //nolint:wrapcheck
}

func (m *mockRemoteConfigPersistence) ListAgentRemoteConfigs(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.AgentRemoteConfig], error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck
	}

	result, ok := args.Get(0).(*model.ListResponse[*agentmodel.AgentRemoteConfig])
	if !ok {
		return nil, errUnexpectedType
	}

	return result, args.Error(1) //nolint:wrapcheck
}

// mockCertPersistence is a mock for CertificatePersistencePort.
type mockCertPersistence struct {
	mock.Mock
}

func (m *mockCertPersistence) GetCertificate(
	ctx context.Context,
	namespace string,
	name string,
	options *model.GetOptions,
) (*agentmodel.Certificate, error) {
	args := m.Called(ctx, namespace, name, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck
	}

	cert, ok := args.Get(0).(*agentmodel.Certificate)
	if !ok {
		return nil, errUnexpectedType
	}

	return cert, args.Error(1) //nolint:wrapcheck
}

func (m *mockCertPersistence) PutCertificate(
	ctx context.Context,
	certificate *agentmodel.Certificate,
) (*agentmodel.Certificate, error) {
	args := m.Called(ctx, certificate)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck
	}

	cert, ok := args.Get(0).(*agentmodel.Certificate)
	if !ok {
		return nil, errUnexpectedType
	}

	return cert, args.Error(1) //nolint:wrapcheck
}

func (m *mockCertPersistence) ListCertificate(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Certificate], error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck
	}

	resp, ok := args.Get(0).(*model.ListResponse[*agentmodel.Certificate])
	if !ok {
		return nil, errUnexpectedType
	}

	return resp, args.Error(1) //nolint:wrapcheck
}

var errUnexpectedType = errors.New("unexpected type")

func TestResolveRemoteConfig_RefMode(t *testing.T) {
	t.Parallel()

	t.Run("Successfully resolves referenced AgentRemoteConfig", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockPersistence := new(mockAgentGroupPersistence)
		mockAgentUC := new(mockAgentUsecase)
		mockRemoteConfigPort := new(mockRemoteConfigPersistence)
		logger := slog.Default()

		mockCertPort := new(mockCertPersistence)
		svc := NewAgentGroupService(mockPersistence, mockRemoteConfigPort, mockCertPort, mockAgentUC, logger)

		refName := "shared-otel-config"
		referencedConfig := &agentmodel.AgentRemoteConfig{
			Metadata: agentmodel.AgentRemoteConfigMetadata{
				Name: refName,
			},
			Spec: agentmodel.AgentRemoteConfigSpec{
				Value:       []byte("receivers:\n  otlp:\n    protocols:\n      grpc:"),
				ContentType: "application/yaml",
			},
		}

		mockRemoteConfigPort.On("GetAgentRemoteConfig", ctx, "", refName, (*model.GetOptions)(nil)).
			Return(referencedConfig, nil)

		remoteConfig := agentmodel.AgentGroupAgentRemoteConfig{
			AgentRemoteConfigRef: &refName,
		}

		configFile, configName, err := svc.resolveRemoteConfig(ctx, "", "test-group", remoteConfig)

		require.NoError(t, err)
		assert.Equal(t, refName, configName) // No prefix for refs
		assert.Equal(t, referencedConfig.Spec.Value, configFile.Body)
		assert.Equal(t, referencedConfig.Spec.ContentType, configFile.ContentType)
		mockRemoteConfigPort.AssertExpectations(t)
	})

	t.Run("Returns error when referenced config not found", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockPersistence := new(mockAgentGroupPersistence)
		mockAgentUC := new(mockAgentUsecase)
		mockRemoteConfigPort := new(mockRemoteConfigPersistence)
		logger := slog.Default()

		mockCertPort := new(mockCertPersistence)
		svc := NewAgentGroupService(mockPersistence, mockRemoteConfigPort, mockCertPort, mockAgentUC, logger)

		refName := "non-existent-config"
		mockRemoteConfigPort.On("GetAgentRemoteConfig", ctx, "", refName, (*model.GetOptions)(nil)).
			Return(nil, errRemoteConfigNotFound)

		remoteConfig := agentmodel.AgentGroupAgentRemoteConfig{
			AgentRemoteConfigRef: &refName,
		}

		_, _, err := svc.resolveRemoteConfig(ctx, "", "test-group", remoteConfig)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "get agent remote config")
		mockRemoteConfigPort.AssertExpectations(t)
	})
}

func TestResolveRemoteConfig_DirectMode(t *testing.T) {
	t.Parallel()

	t.Run("Inline config gets AgentGroupName prefix", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockPersistence := new(mockAgentGroupPersistence)
		mockAgentUC := new(mockAgentUsecase)
		mockRemoteConfigPort := new(mockRemoteConfigPersistence)
		logger := slog.Default()

		mockCertPort := new(mockCertPersistence)
		svc := NewAgentGroupService(mockPersistence, mockRemoteConfigPort, mockCertPort, mockAgentUC, logger)

		configName := "collector-config"
		configValue := []byte("exporters:\n  debug:\n    verbosity: detailed")
		contentType := "application/yaml"

		remoteConfig := agentmodel.AgentGroupAgentRemoteConfig{
			AgentRemoteConfigName: &configName,
			AgentRemoteConfigSpec: &agentmodel.AgentRemoteConfigSpec{
				Value:       configValue,
				ContentType: contentType,
			},
		}

		configFile, resolvedName, err := svc.resolveRemoteConfig(ctx, "", "staging-group", remoteConfig)

		require.NoError(t, err)
		// Config name should be prefixed with AgentGroupName
		assert.Equal(t, "staging-group/collector-config", resolvedName)
		assert.Equal(t, configValue, configFile.Body)
		assert.Equal(t, contentType, configFile.ContentType)
	})

	t.Run("Returns error when spec is nil", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockPersistence := new(mockAgentGroupPersistence)
		mockAgentUC := new(mockAgentUsecase)
		mockRemoteConfigPort := new(mockRemoteConfigPersistence)
		logger := slog.Default()

		mockCertPort := new(mockCertPersistence)
		svc := NewAgentGroupService(mockPersistence, mockRemoteConfigPort, mockCertPort, mockAgentUC, logger)

		configName := "missing-spec-config"
		remoteConfig := agentmodel.AgentGroupAgentRemoteConfig{
			AgentRemoteConfigName: &configName,
			AgentRemoteConfigSpec: nil, // Missing spec
		}

		_, _, err := svc.resolveRemoteConfig(ctx, "", "test-group", remoteConfig)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidRemoteConfig)
	})

	t.Run("Returns error when name is nil", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockPersistence := new(mockAgentGroupPersistence)
		mockAgentUC := new(mockAgentUsecase)
		mockRemoteConfigPort := new(mockRemoteConfigPersistence)
		logger := slog.Default()

		mockCertPort := new(mockCertPersistence)
		svc := NewAgentGroupService(mockPersistence, mockRemoteConfigPort, mockCertPort, mockAgentUC, logger)

		remoteConfig := agentmodel.AgentGroupAgentRemoteConfig{
			AgentRemoteConfigName: nil, // Missing name
			AgentRemoteConfigSpec: &agentmodel.AgentRemoteConfigSpec{
				Value:       []byte("some config"),
				ContentType: "text/plain",
			},
		}

		_, _, err := svc.resolveRemoteConfig(ctx, "", "test-group", remoteConfig)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidRemoteConfig)
	})
}

func TestApplyRemoteConfigs(t *testing.T) {
	t.Parallel()

	t.Run("Applies ref config to agent without prefix", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockPersistence := new(mockAgentGroupPersistence)
		mockAgentUC := new(mockAgentUsecase)
		mockRemoteConfigPort := new(mockRemoteConfigPersistence)
		logger := slog.Default()

		mockCertPort := new(mockCertPersistence)
		svc := NewAgentGroupService(mockPersistence, mockRemoteConfigPort, mockCertPort, mockAgentUC, logger)

		testAgent := agentmodel.NewAgent(uuid.New(), agentmodel.WithDescription(&agent.Description{
			IdentifyingAttributes: map[string]string{"service.name": "test"},
		}))

		refName := "global-config"
		referencedConfig := &agentmodel.AgentRemoteConfig{
			Metadata: agentmodel.AgentRemoteConfigMetadata{Name: refName},
			Spec: agentmodel.AgentRemoteConfigSpec{
				Value:       []byte("global config content"),
				ContentType: "text/plain",
			},
		}

		agentGroup := &agentmodel.AgentGroup{
			Metadata: agentmodel.AgentGroupMetadata{Name: "production"},
			Spec: agentmodel.AgentGroupSpec{
				AgentRemoteConfigs: []agentmodel.AgentGroupAgentRemoteConfig{
					{AgentRemoteConfigRef: &refName},
				},
			},
		}

		mockRemoteConfigPort.On("GetAgentRemoteConfig", ctx, "", refName, (*model.GetOptions)(nil)).
			Return(referencedConfig, nil)

		resolved, err := svc.collectGroupRemoteConfigs(ctx, agentGroup)

		require.NoError(t, err)
		// Verify config was resolved under its original name (no prefix)
		configFile, exists := resolved[refName]
		assert.True(t, exists, "Config should be resolved under its original name")
		assert.Equal(t, referencedConfig.Spec.Value, configFile.Body)
		mockRemoteConfigPort.AssertExpectations(t)

		_ = testAgent
	})

	t.Run("Applies inline config to agent with AgentGroupName prefix", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockPersistence := new(mockAgentGroupPersistence)
		mockAgentUC := new(mockAgentUsecase)
		mockRemoteConfigPort := new(mockRemoteConfigPersistence)
		logger := slog.Default()

		mockCertPort := new(mockCertPersistence)
		svc := NewAgentGroupService(mockPersistence, mockRemoteConfigPort, mockCertPort, mockAgentUC, logger)

		testAgent := agentmodel.NewAgent(uuid.New(), agentmodel.WithDescription(&agent.Description{
			IdentifyingAttributes: map[string]string{"service.name": "test"},
		}))

		inlineName := "local-config"
		inlineValue := []byte("local config content")
		agentGroup := &agentmodel.AgentGroup{
			Metadata: agentmodel.AgentGroupMetadata{Name: "staging"},
			Spec: agentmodel.AgentGroupSpec{
				AgentRemoteConfigs: []agentmodel.AgentGroupAgentRemoteConfig{
					{
						AgentRemoteConfigName: &inlineName,
						AgentRemoteConfigSpec: &agentmodel.AgentRemoteConfigSpec{
							Value:       inlineValue,
							ContentType: "text/plain",
						},
					},
				},
			},
		}

		resolved, err := svc.collectGroupRemoteConfigs(ctx, agentGroup)

		require.NoError(t, err)
		// Verify config was resolved under its prefixed name
		expectedName := "staging/local-config"
		configFile, exists := resolved[expectedName]
		assert.True(t, exists, "Config should be resolved under prefixed name: %s", expectedName)
		assert.Equal(t, inlineValue, configFile.Body)

		_ = testAgent
	})
}

func TestNameCollisionPrevention(t *testing.T) {
	t.Parallel()

	t.Run("Same config name in different groups produces different keys", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockPersistence := new(mockAgentGroupPersistence)
		mockAgentUC := new(mockAgentUsecase)
		mockRemoteConfigPort := new(mockRemoteConfigPersistence)
		logger := slog.Default()

		mockCertPort := new(mockCertPersistence)
		svc := NewAgentGroupService(mockPersistence, mockRemoteConfigPort, mockCertPort, mockAgentUC, logger)

		// Create agent
		testAgent := agentmodel.NewAgent(uuid.New(), agentmodel.WithDescription(&agent.Description{
			IdentifyingAttributes: map[string]string{"service.name": "test"},
		}))

		configName := "config" // Same name used in both groups

		// Group Alpha
		groupAlpha := &agentmodel.AgentGroup{
			Metadata: agentmodel.AgentGroupMetadata{Name: "group-alpha"},
			Spec: agentmodel.AgentGroupSpec{
				AgentRemoteConfigs: []agentmodel.AgentGroupAgentRemoteConfig{
					{
						AgentRemoteConfigName: &configName,
						AgentRemoteConfigSpec: &agentmodel.AgentRemoteConfigSpec{
							Value:       []byte("content from alpha"),
							ContentType: "text/plain",
						},
					},
				},
			},
		}

		// Group Beta
		groupBeta := &agentmodel.AgentGroup{
			Metadata: agentmodel.AgentGroupMetadata{Name: "group-beta"},
			Spec: agentmodel.AgentGroupSpec{
				AgentRemoteConfigs: []agentmodel.AgentGroupAgentRemoteConfig{
					{
						AgentRemoteConfigName: &configName,
						AgentRemoteConfigSpec: &agentmodel.AgentRemoteConfigSpec{
							Value:       []byte("content from beta"),
							ContentType: "text/plain",
						},
					},
				},
			},
		}

		// Resolve configs from both groups
		alphaResolved, err := svc.collectGroupRemoteConfigs(ctx, groupAlpha)
		require.NoError(t, err)

		betaResolved, err := svc.collectGroupRemoteConfigs(ctx, groupBeta)
		require.NoError(t, err)

		// Verify both configs exist with different prefixed names
		alphaConfig, alphaExists := alphaResolved["group-alpha/config"]
		betaConfig, betaExists := betaResolved["group-beta/config"]

		assert.True(t, alphaExists, "Alpha config should exist")
		assert.True(t, betaExists, "Beta config should exist")
		assert.Equal(t, []byte("content from alpha"), alphaConfig.Body)
		assert.Equal(t, []byte("content from beta"), betaConfig.Body)

		// Verify each group resolves to exactly its own config (no collision when keyed by prefixed name)
		assert.Len(t, alphaResolved, 1)
		assert.Len(t, betaResolved, 1)

		_ = testAgent
	})
}

func TestRecordRemoteConfigCondition(t *testing.T) {
	t.Parallel()

	newSvc := func() (*AgentGroupService, *mockAgentGroupPersistence) {
		mockPersistence := new(mockAgentGroupPersistence)
		svc := NewAgentGroupService(
			mockPersistence,
			new(mockRemoteConfigPersistence),
			new(mockCertPersistence),
			new(mockAgentUsecase),
			slog.Default(),
		)

		return svc, mockPersistence
	}

	t.Run("records False condition with error message when inline config is invalid", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		svc, mockPersistence := newSvc()

		// Inline config missing AgentRemoteConfigName -> ErrInvalidRemoteConfig.
		inlineSpec := &agentmodel.AgentRemoteConfigSpec{Value: []byte("x"), ContentType: "text/plain"}
		group := &agentmodel.AgentGroup{
			Metadata: agentmodel.AgentGroupMetadata{Namespace: "default", Name: "broken"},
			Spec: agentmodel.AgentGroupSpec{
				AgentRemoteConfigs: []agentmodel.AgentGroupAgentRemoteConfig{
					{AgentRemoteConfigSpec: inlineSpec},
				},
			},
		}

		// recordRemoteConfigCondition re-reads the group before writing the condition.
		mockPersistence.On("GetAgentGroup", mock.Anything, "default", "broken", (*model.GetOptions)(nil)).
			Return(group, nil)
		mockPersistence.On("PutAgentGroup", mock.Anything, "default", "broken", mock.Anything).
			Return(group, nil)

		err := svc.recordRemoteConfigCondition(ctx, group)

		require.ErrorIs(t, err, ErrInvalidRemoteConfig)

		cond := group.GetCondition(model.ConditionTypeRemoteConfigApplied)
		require.NotNil(t, cond)
		assert.Equal(t, model.ConditionStatusFalse, cond.Status)
		assert.Contains(t, cond.Message, "invalid remote config")
		mockPersistence.AssertExpectations(t)
	})

	t.Run("records True condition when inline config resolves", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		svc, mockPersistence := newSvc()

		name := "ok-config"
		group := &agentmodel.AgentGroup{
			Metadata: agentmodel.AgentGroupMetadata{Namespace: "default", Name: "good"},
			Spec: agentmodel.AgentGroupSpec{
				AgentRemoteConfigs: []agentmodel.AgentGroupAgentRemoteConfig{
					{
						AgentRemoteConfigName: &name,
						AgentRemoteConfigSpec: &agentmodel.AgentRemoteConfigSpec{
							Value:       []byte("receivers: {}"),
							ContentType: "text/yaml",
						},
					},
				},
			},
		}

		// recordRemoteConfigCondition re-reads the group before writing the condition.
		mockPersistence.On("GetAgentGroup", mock.Anything, "default", "good", (*model.GetOptions)(nil)).
			Return(group, nil)
		mockPersistence.On("PutAgentGroup", mock.Anything, "default", "good", mock.Anything).
			Return(group, nil)

		err := svc.recordRemoteConfigCondition(ctx, group)

		require.NoError(t, err)

		cond := group.GetCondition(model.ConditionTypeRemoteConfigApplied)
		require.NotNil(t, cond)
		assert.Equal(t, model.ConditionStatusTrue, cond.Status)
		mockPersistence.AssertExpectations(t)
	})

	t.Run("skips groups that declare no remote config", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		svc, mockPersistence := newSvc()

		group := &agentmodel.AgentGroup{
			Metadata: agentmodel.AgentGroupMetadata{Namespace: "default", Name: "no-config"},
			Spec:     agentmodel.AgentGroupSpec{},
		}

		err := svc.recordRemoteConfigCondition(ctx, group)

		require.NoError(t, err)
		assert.Nil(t, group.GetCondition(model.ConditionTypeRemoteConfigApplied))
		// No remote config declared -> nothing to record -> no persistence write.
		mockPersistence.AssertNotCalled(t, "PutAgentGroup", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("does not re-persist when condition is unchanged", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		svc, mockPersistence := newSvc()

		name := "ok-config"
		group := &agentmodel.AgentGroup{
			Metadata: agentmodel.AgentGroupMetadata{Namespace: "default", Name: "good"},
			Spec: agentmodel.AgentGroupSpec{
				AgentRemoteConfigs: []agentmodel.AgentGroupAgentRemoteConfig{
					{
						AgentRemoteConfigName: &name,
						AgentRemoteConfigSpec: &agentmodel.AgentRemoteConfigSpec{
							Value:       []byte("receivers: {}"),
							ContentType: "text/yaml",
						},
					},
				},
			},
		}

		// The re-read returns the same object both times; after the first write it carries
		// the condition, so the second call sees no change.
		mockPersistence.On("GetAgentGroup", mock.Anything, "default", "good", (*model.GetOptions)(nil)).
			Return(group, nil)
		// Only the first call should persist; the second sees an unchanged condition.
		mockPersistence.On("PutAgentGroup", mock.Anything, "default", "good", mock.Anything).
			Return(group, nil).Once()

		require.NoError(t, svc.recordRemoteConfigCondition(ctx, group))
		require.NoError(t, svc.recordRemoteConfigCondition(ctx, group))

		mockPersistence.AssertExpectations(t)
	})
}

func TestRecordAgentRemoteConfigCondition(t *testing.T) {
	t.Parallel()

	svc := NewAgentGroupService(
		new(mockAgentGroupPersistence),
		new(mockRemoteConfigPersistence),
		new(mockCertPersistence),
		new(mockAgentUsecase),
		slog.Default(),
	)

	group := &agentmodel.AgentGroup{
		Metadata: agentmodel.AgentGroupMetadata{Namespace: "default", Name: "grp"},
	}

	withAssignedConfig := func(a *agentmodel.Agent) {
		a.Spec.RemoteConfig = &agentmodel.AgentSpecRemoteConfig{
			ConfigMap: agentmodel.AgentConfigMap{
				ConfigMap: map[string]agentmodel.AgentConfigFile{"grp/cfg": {}},
			},
		}
	}

	t.Run("config-accepting agent gets True", func(t *testing.T) {
		t.Parallel()

		a := agentmodel.NewAgent(uuid.New())
		a.Metadata.Capabilities = agent.Capabilities(agent.AgentCapabilityAcceptsRemoteConfig)
		withAssignedConfig(a)

		changed := svc.recordAgentRemoteConfigCondition(a, group)
		assert.True(t, changed)

		cond := a.GetCondition(agentmodel.AgentConditionTypeRemoteConfigApplied)
		require.NotNil(t, cond)
		assert.Equal(t, agentmodel.AgentConditionStatusTrue, cond.Status)
	})

	t.Run("agent without AcceptsRemoteConfig gets False with explanation", func(t *testing.T) {
		t.Parallel()

		a := agentmodel.NewAgent(uuid.New()) // default capabilities: none
		withAssignedConfig(a)

		changed := svc.recordAgentRemoteConfigCondition(a, group)
		assert.True(t, changed)

		cond := a.GetCondition(agentmodel.AgentConditionTypeRemoteConfigApplied)
		require.NotNil(t, cond)
		assert.Equal(t, agentmodel.AgentConditionStatusFalse, cond.Status)
		assert.Contains(t, cond.Message, "does not accept remote config")
	})

	t.Run("no assigned config leaves the condition untouched", func(t *testing.T) {
		t.Parallel()

		a := agentmodel.NewAgent(uuid.New())

		changed := svc.recordAgentRemoteConfigCondition(a, group)

		assert.False(t, changed)
		assert.Nil(t, a.GetCondition(agentmodel.AgentConditionTypeRemoteConfigApplied))
	})

	t.Run("unchanged condition reports no change on repeat", func(t *testing.T) {
		t.Parallel()

		a := agentmodel.NewAgent(uuid.New())
		a.Metadata.Capabilities = agent.Capabilities(agent.AgentCapabilityAcceptsRemoteConfig)
		withAssignedConfig(a)

		assert.True(t, svc.recordAgentRemoteConfigCondition(a, group))
		assert.False(t, svc.recordAgentRemoteConfigCondition(a, group))
	})
}

func TestDeleteAgentGroup_PropagatesDeletion(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockPersistence := new(mockAgentGroupPersistence)
	svc := NewAgentGroupService(
		mockPersistence,
		new(mockRemoteConfigPersistence),
		new(mockCertPersistence),
		new(mockAgentUsecase),
		slog.Default(),
	)

	existing := &agentmodel.AgentGroup{
		Metadata: agentmodel.AgentGroupMetadata{Namespace: "default", Name: "to-delete"},
		Spec: agentmodel.AgentGroupSpec{
			Selector: agentmodel.AgentSelector{
				IdentifyingAttributes: map[string]string{"service.name": "my-service"},
			},
		},
	}

	mockPersistence.On("GetAgentGroup", ctx, "default", "to-delete", (*model.GetOptions)(nil)).
		Return(existing, nil)
	mockPersistence.On("PutAgentGroup", ctx, "default", "to-delete", mock.Anything).
		Return(existing, nil)

	err := svc.DeleteAgentGroup(ctx, "default", "to-delete", time.Date(2026, time.May, 31, 12, 0, 0, 0, time.UTC), "admin")
	require.NoError(t, err)

	// The deletion must be queued so former members get the group's config dropped.
	select {
	case queued := <-svc.changedAgentGroupCh:
		assert.True(t, queued.IsDeleted(), "queued group should be marked deleted")
		assert.Equal(t, "to-delete", queued.Metadata.Name)
	default:
		t.Fatal("DeleteAgentGroup did not propagate the deletion to agents")
	}

	mockPersistence.AssertExpectations(t)
}

func TestShouldReconcileDeletedGroup(t *testing.T) {
	t.Parallel()

	// Uses the default real clock; offsets are far larger than the test runtime so the
	// in/out-of-window comparisons are not racy.
	svc := NewAgentGroupService(
		new(mockAgentGroupPersistence),
		new(mockRemoteConfigPersistence),
		new(mockCertPersistence),
		new(mockAgentUsecase),
		slog.Default(),
	)

	newDeletedGroup := func(deletedAt time.Time) *agentmodel.AgentGroup {
		group := &agentmodel.AgentGroup{
			Metadata: agentmodel.AgentGroupMetadata{Namespace: "default", Name: "g"},
			Spec:     agentmodel.AgentGroupSpec{},
		}
		group.MarkDeleted(deletedAt, "admin")

		return group
	}

	t.Run("within window", func(t *testing.T) {
		t.Parallel()

		group := newDeletedGroup(time.Now().Add(-DeletedGroupReconcileWindow / 2))
		assert.True(t, svc.shouldReconcileDeletedGroup(group))
	})

	t.Run("outside window", func(t *testing.T) {
		t.Parallel()

		group := newDeletedGroup(time.Now().Add(-2 * DeletedGroupReconcileWindow))
		assert.False(t, svc.shouldReconcileDeletedGroup(group))
	})
}

func TestUpdateAgentsByAgentGroup(t *testing.T) {
	t.Parallel()

	t.Run("Full propagation flow with ref config", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockPersistence := new(mockAgentGroupPersistence)
		mockAgentUC := new(mockAgentUsecase)
		mockRemoteConfigPort := new(mockRemoteConfigPersistence)
		logger := slog.Default()

		mockCertPort := new(mockCertPersistence)
		svc := NewAgentGroupService(mockPersistence, mockRemoteConfigPort, mockCertPort, mockAgentUC, logger)

		testAgent := agentmodel.NewAgent(uuid.New(), agentmodel.WithDescription(&agent.Description{
			IdentifyingAttributes: map[string]string{"service.name": "my-service"},
		}))

		refName := "shared-config"
		referencedConfig := &agentmodel.AgentRemoteConfig{
			Metadata: agentmodel.AgentRemoteConfigMetadata{Name: refName},
			Spec: agentmodel.AgentRemoteConfigSpec{
				Value:       []byte("shared config content"),
				ContentType: "application/yaml",
			},
		}

		agentGroup := &agentmodel.AgentGroup{
			Metadata: agentmodel.AgentGroupMetadata{
				Name: "production",
			},
			Spec: agentmodel.AgentGroupSpec{
				Selector: agentmodel.AgentSelector{
					IdentifyingAttributes: map[string]string{"service.name": "my-service"},
				},
				AgentRemoteConfigs: []agentmodel.AgentGroupAgentRemoteConfig{
					{AgentRemoteConfigRef: &refName},
				},
			},
		}

		agentsResponse := &model.ListResponse[*agentmodel.Agent]{
			Items:              []*agentmodel.Agent{testAgent},
			Continue:           "",
			RemainingItemCount: 0,
		}

		mockAgentUC.On("ListAgentsBySelector", ctx, agentGroup.Spec.Selector, mock.Anything).
			Return(agentsResponse, nil)
		// updateAgentsByAgentGroup now applies the union of all matching groups per agent
		// (ApplyMatchingAgentGroupsToAgent), which calls GetAgentGroupsForAgent → ListAgentGroups.
		mockPersistence.On("ListAgentGroups", mock.Anything, (*model.ListOptions)(nil)).
			Return(&model.ListResponse[*agentmodel.AgentGroup]{Items: []*agentmodel.AgentGroup{agentGroup}}, nil)
		mockRemoteConfigPort.On("GetAgentRemoteConfig", mock.Anything, "", refName, (*model.GetOptions)(nil)).
			Return(referencedConfig, nil)
		// updateAgentsByAgentGroup records the RemoteConfigApplied condition on the group;
		// recordRemoteConfigCondition re-reads it first, then persists.
		mockPersistence.On("GetAgentGroup", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(agentGroup, nil)
		mockPersistence.On("PutAgentGroup", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(agentGroup, nil)
		mockAgentUC.On("SaveAgent", ctx, mock.MatchedBy(func(a *agentmodel.Agent) bool {
			_, exists := a.Spec.RemoteConfig.ConfigMap.ConfigMap[refName]

			return exists
		})).Return(nil)

		err := svc.updateAgentsByAgentGroup(ctx, agentGroup)

		require.NoError(t, err)
		mockAgentUC.AssertExpectations(t)
		mockRemoteConfigPort.AssertExpectations(t)
	})

	t.Run("Full propagation flow with inline config", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockPersistence := new(mockAgentGroupPersistence)
		mockAgentUC := new(mockAgentUsecase)
		mockRemoteConfigPort := new(mockRemoteConfigPersistence)
		logger := slog.Default()

		mockCertPort := new(mockCertPersistence)
		svc := NewAgentGroupService(mockPersistence, mockRemoteConfigPort, mockCertPort, mockAgentUC, logger)

		testAgent := agentmodel.NewAgent(uuid.New(), agentmodel.WithDescription(&agent.Description{
			IdentifyingAttributes: map[string]string{"service.name": "my-service"},
		}))

		inlineName := "inline-config"
		agentGroup := &agentmodel.AgentGroup{
			Metadata: agentmodel.AgentGroupMetadata{
				Name: "staging",
			},
			Spec: agentmodel.AgentGroupSpec{
				Selector: agentmodel.AgentSelector{
					IdentifyingAttributes: map[string]string{"service.name": "my-service"},
				},
				AgentRemoteConfigs: []agentmodel.AgentGroupAgentRemoteConfig{
					{
						AgentRemoteConfigName: &inlineName,
						AgentRemoteConfigSpec: &agentmodel.AgentRemoteConfigSpec{
							Value:       []byte("inline config content"),
							ContentType: "text/plain",
						},
					},
				},
			},
		}

		agentsResponse := &model.ListResponse[*agentmodel.Agent]{
			Items:              []*agentmodel.Agent{testAgent},
			Continue:           "",
			RemainingItemCount: 0,
		}

		mockAgentUC.On("ListAgentsBySelector", ctx, agentGroup.Spec.Selector, mock.Anything).
			Return(agentsResponse, nil)
		// updateAgentsByAgentGroup → ApplyMatchingAgentGroupsToAgent → GetAgentGroupsForAgent.
		mockPersistence.On("ListAgentGroups", mock.Anything, (*model.ListOptions)(nil)).
			Return(&model.ListResponse[*agentmodel.AgentGroup]{Items: []*agentmodel.AgentGroup{agentGroup}}, nil)
		// updateAgentsByAgentGroup records the RemoteConfigApplied condition on the group;
		// recordRemoteConfigCondition re-reads it first, then persists.
		mockPersistence.On("GetAgentGroup", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(agentGroup, nil)
		mockPersistence.On("PutAgentGroup", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(agentGroup, nil)
		mockAgentUC.On("SaveAgent", ctx, mock.MatchedBy(func(a *agentmodel.Agent) bool {
			// Verify the config has the prefixed name
			_, exists := a.Spec.RemoteConfig.ConfigMap.ConfigMap["staging/inline-config"]

			return exists
		})).Return(nil)

		err := svc.updateAgentsByAgentGroup(ctx, agentGroup)

		require.NoError(t, err)
		mockAgentUC.AssertExpectations(t)
	})
}
