package agentservice

import (
	"context"
	"errors"
	"log/slog"
	"testing"

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
	name string,
	options *model.GetOptions,
) (*agentmodel.AgentGroup, error) {
	args := m.Called(ctx, name, options)
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
	name string,
	ag *agentmodel.AgentGroup,
) (*agentmodel.AgentGroup, error) {
	args := m.Called(ctx, name, ag)
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

func (m *mockAgentUsecase) ListAgents(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Agent], error) {
	args := m.Called(ctx, options)
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
	query string,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Agent], error) {
	args := m.Called(ctx, query, options)
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
	name string,
) (*agentmodel.AgentRemoteConfig, error) {
	args := m.Called(ctx, name)
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
	name string,
) (*agentmodel.Certificate, error) {
	args := m.Called(ctx, name)
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

		mockRemoteConfigPort.On("GetAgentRemoteConfig", ctx, refName).Return(referencedConfig, nil)

		remoteConfig := agentmodel.AgentGroupAgentRemoteConfig{
			AgentRemoteConfigRef: &refName,
		}

		configFile, configName, err := svc.resolveRemoteConfig(ctx, "test-group", remoteConfig)

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
		mockRemoteConfigPort.On("GetAgentRemoteConfig", ctx, refName).Return(nil, errRemoteConfigNotFound)

		remoteConfig := agentmodel.AgentGroupAgentRemoteConfig{
			AgentRemoteConfigRef: &refName,
		}

		_, _, err := svc.resolveRemoteConfig(ctx, "test-group", remoteConfig)

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

		configFile, resolvedName, err := svc.resolveRemoteConfig(ctx, "staging-group", remoteConfig)

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

		_, _, err := svc.resolveRemoteConfig(ctx, "test-group", remoteConfig)

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

		_, _, err := svc.resolveRemoteConfig(ctx, "test-group", remoteConfig)

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

		mockRemoteConfigPort.On("GetAgentRemoteConfig", ctx, refName).Return(referencedConfig, nil)

		err := svc.applyRemoteConfigs(ctx, agentGroup, testAgent)

		require.NoError(t, err)
		// Verify config was applied with original name (no prefix)
		configFile, exists := testAgent.Spec.RemoteConfig.ConfigMap.ConfigMap[refName]
		assert.True(t, exists, "Config should be applied with original name")
		assert.Equal(t, referencedConfig.Spec.Value, configFile.Body)
		mockRemoteConfigPort.AssertExpectations(t)
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

		err := svc.applyRemoteConfigs(ctx, agentGroup, testAgent)

		require.NoError(t, err)
		// Verify config was applied with prefixed name
		expectedName := "staging/local-config"
		configFile, exists := testAgent.Spec.RemoteConfig.ConfigMap.ConfigMap[expectedName]
		assert.True(t, exists, "Config should be applied with prefixed name: %s", expectedName)
		assert.Equal(t, inlineValue, configFile.Body)
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

		// Apply configs from both groups
		err := svc.applyRemoteConfigs(ctx, groupAlpha, testAgent)
		require.NoError(t, err)

		err = svc.applyRemoteConfigs(ctx, groupBeta, testAgent)
		require.NoError(t, err)

		// Verify both configs exist with different prefixed names
		alphaConfig, alphaExists := testAgent.Spec.RemoteConfig.ConfigMap.ConfigMap["group-alpha/config"]
		betaConfig, betaExists := testAgent.Spec.RemoteConfig.ConfigMap.ConfigMap["group-beta/config"]

		assert.True(t, alphaExists, "Alpha config should exist")
		assert.True(t, betaExists, "Beta config should exist")
		assert.Equal(t, []byte("content from alpha"), alphaConfig.Body)
		assert.Equal(t, []byte("content from beta"), betaConfig.Body)

		// Verify we have exactly 2 configs (no collision)
		assert.Len(t, testAgent.Spec.RemoteConfig.ConfigMap.ConfigMap, 2)
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
		mockRemoteConfigPort.On("GetAgentRemoteConfig", ctx, refName).Return(referencedConfig, nil)
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
