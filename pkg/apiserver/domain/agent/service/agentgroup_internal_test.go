package agentservice

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"log/slog"
	"math/big"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	inmemorypersistence "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/inmemory"
	inmemorystore "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/store/inmemory"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
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
	namespace string,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.AgentGroup], error) {
	args := m.Called(ctx, namespace, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck
	}

	result, ok := args.Get(0).(*model.ListResponse[*agentmodel.AgentGroup])
	if !ok {
		return nil, errUnexpectedType
	}

	return result, args.Error(1) //nolint:wrapcheck
}

// ListAllAgentGroups records the cluster-wide listing separately, so a test can
// tell "every namespace" apart from a scoped one rather than matching on "".
func (m *mockAgentGroupPersistence) ListAllAgentGroups(
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

// TouchAgentLiveness is unused by these tests: the liveness fast tier is exercised
// in the agent service's own tests.
func (m *mockAgentUsecase) TouchAgentLiveness(
	_ context.Context,
	_ *agentmodel.Agent,
	_ time.Time,
) bool {
	return false
}

// ForgetAgentLiveness is unused by these tests.
func (m *mockAgentUsecase) ForgetAgentLiveness(_ context.Context, _ uuid.UUID) error {
	return nil
}

// PersistAgentLiveness is unused by these tests.
func (m *mockAgentUsecase) PersistAgentLiveness(_ context.Context, _ *agentmodel.AgentLiveness) error {
	return nil
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
	namespace string,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.AgentRemoteConfig], error) {
	args := m.Called(ctx, namespace, options)
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
	namespace string,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Certificate], error) {
	args := m.Called(ctx, namespace, options)
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

func opampTestCertificate(t *testing.T, commonName string) *agentmodel.Certificate {
	t.Helper()

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	der, err := x509.CreateCertificate(rand.Reader, &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: commonName},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
	}, &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: commonName},
	}, publicKey, privateKey)
	require.NoError(t, err)
	keyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)

	return &agentmodel.Certificate{Spec: agentmodel.CertificateSpec{
		Cert:       pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		PrivateKey: pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}),
	}}
}

func TestSaveAgentGroupRejectsUnsafeOpAMPCertificate(t *testing.T) {
	t.Parallel()

	uid := uuid.New()
	name := "next-cert"
	group := &agentmodel.AgentGroup{
		Metadata: agentmodel.AgentGroupMetadata{Name: "group", Namespace: "default"},
		Spec: agentmodel.AgentGroupSpec{AgentConnectionConfig: &agentmodel.AgentGroupConnectionConfig{
			OpAMPConnection: &agentmodel.OpAMPConnectionSettings{
				DestinationEndpoint: "wss://example.test", CertificateName: &name,
			},
		}},
	}

	for _, testCase := range []struct {
		name        string
		members     *model.ListResponse[*agentmodel.Agent]
		certificate *agentmodel.Certificate
	}{
		{name: "multiple agents", members: &model.ListResponse[*agentmodel.Agent]{
			Items: []*agentmodel.Agent{agentmodel.NewAgent(uid), agentmodel.NewAgent(uuid.New())},
		}},
		{name: "invalid key pair", members: &model.ListResponse[*agentmodel.Agent]{
			Items: []*agentmodel.Agent{agentmodel.NewAgent(uid)},
		}, certificate: &agentmodel.Certificate{Spec: agentmodel.CertificateSpec{
			Cert: []byte("bad"), PrivateKey: []byte("bad"),
		}}},
		{name: "wrong CN", members: &model.ListResponse[*agentmodel.Agent]{
			Items: []*agentmodel.Agent{agentmodel.NewAgent(uid)},
		}, certificate: opampTestCertificate(t, uuid.NewString())},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			groups := new(mockAgentGroupPersistence)
			agents := new(mockAgentUsecase)
			certs := new(mockCertPersistence)
			svc := NewAgentGroupService(
				groups,
				new(mockRemoteConfigPersistence),
				certs,
				agents,
				alwaysLeaderElector{},
				inmemorystore.NewAgentGroupChangeStore(ChangedAgentGroupBufferSize),
				slog.Default(),
			)
			agents.On("ListAgentsBySelector", ctx, group.Spec.Selector, mock.Anything).
				Return(testCase.members, nil)

			if testCase.certificate != nil {
				certs.On("GetCertificate", ctx, "default", name, (*model.GetOptions)(nil)).
					Return(testCase.certificate, nil)
			}

			_, err := svc.SaveAgentGroup(ctx, "default", "group", group)
			require.ErrorIs(t, err, model.ErrInvalidArgument)
			groups.AssertNotCalled(t, "PutAgentGroup", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestApplyConnectionSettingsSkipsCertificateForDifferentAgent(t *testing.T) {
	t.Parallel()

	uid := uuid.New()
	name := "next-cert"
	group := &agentmodel.AgentGroup{
		Metadata: agentmodel.AgentGroupMetadata{Name: "group", Namespace: "default"},
		Spec: agentmodel.AgentGroupSpec{AgentConnectionConfig: &agentmodel.AgentGroupConnectionConfig{
			OpAMPConnection: &agentmodel.OpAMPConnectionSettings{
				DestinationEndpoint: "wss://example.test", CertificateName: &name,
			},
		}},
	}
	agent := agentmodel.NewAgent(uid)
	require.NoError(t, agent.ApplyConnectionSettings(&agentmodel.AgentOpAMPConnectionSettings{
		DestinationEndpoint: "wss://old.example.test",
	}, nil, nil, nil, nil))

	certs := new(mockCertPersistence)
	certs.On("GetCertificate", mock.Anything, "default", name, (*model.GetOptions)(nil)).
		Return(opampTestCertificate(t, uuid.NewString()), nil)
	svc := NewAgentGroupService(
		new(mockAgentGroupPersistence),
		new(mockRemoteConfigPersistence),
		certs,
		new(mockAgentUsecase),
		alwaysLeaderElector{},
		inmemorystore.NewAgentGroupChangeStore(ChangedAgentGroupBufferSize),
		slog.Default(),
	)

	info, err := svc.resolveConnectionSettings(t.Context(), group, agent)
	require.NoError(t, err)
	assert.Nil(t, info)
	assert.Equal(t, "wss://old.example.test", agent.Spec.ConnectionInfo.OpAMP().DestinationEndpoint)
}

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
		svc := NewAgentGroupService(
			mockPersistence,
			mockRemoteConfigPort,
			mockCertPort,
			mockAgentUC,
			alwaysLeaderElector{},
			inmemorystore.NewAgentGroupChangeStore(ChangedAgentGroupBufferSize),
			logger,
		)

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
		svc := NewAgentGroupService(
			mockPersistence,
			mockRemoteConfigPort,
			mockCertPort,
			mockAgentUC,
			alwaysLeaderElector{},
			inmemorystore.NewAgentGroupChangeStore(ChangedAgentGroupBufferSize),
			logger,
		)

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
		svc := NewAgentGroupService(
			mockPersistence,
			mockRemoteConfigPort,
			mockCertPort,
			mockAgentUC,
			alwaysLeaderElector{},
			inmemorystore.NewAgentGroupChangeStore(ChangedAgentGroupBufferSize),
			logger,
		)

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
		svc := NewAgentGroupService(
			mockPersistence,
			mockRemoteConfigPort,
			mockCertPort,
			mockAgentUC,
			alwaysLeaderElector{},
			inmemorystore.NewAgentGroupChangeStore(ChangedAgentGroupBufferSize),
			logger,
		)

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
		svc := NewAgentGroupService(
			mockPersistence,
			mockRemoteConfigPort,
			mockCertPort,
			mockAgentUC,
			alwaysLeaderElector{},
			inmemorystore.NewAgentGroupChangeStore(ChangedAgentGroupBufferSize),
			logger,
		)

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
		svc := NewAgentGroupService(
			mockPersistence,
			mockRemoteConfigPort,
			mockCertPort,
			mockAgentUC,
			alwaysLeaderElector{},
			inmemorystore.NewAgentGroupChangeStore(ChangedAgentGroupBufferSize),
			logger,
		)

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
		svc := NewAgentGroupService(
			mockPersistence,
			mockRemoteConfigPort,
			mockCertPort,
			mockAgentUC,
			alwaysLeaderElector{},
			inmemorystore.NewAgentGroupChangeStore(ChangedAgentGroupBufferSize),
			logger,
		)

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
		svc := NewAgentGroupService(
			mockPersistence,
			mockRemoteConfigPort,
			mockCertPort,
			mockAgentUC,
			alwaysLeaderElector{},
			inmemorystore.NewAgentGroupChangeStore(ChangedAgentGroupBufferSize),
			logger,
		)

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
			alwaysLeaderElector{},
			inmemorystore.NewAgentGroupChangeStore(ChangedAgentGroupBufferSize),
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
		alwaysLeaderElector{},
		inmemorystore.NewAgentGroupChangeStore(ChangedAgentGroupBufferSize),
		slog.Default(),
	)

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

		changed := svc.recordAgentRemoteConfigCondition(a)
		assert.True(t, changed)

		cond := a.GetCondition(agentmodel.AgentConditionTypeRemoteConfigApplied)
		require.NotNil(t, cond)
		assert.Equal(t, agentmodel.AgentConditionStatusTrue, cond.Status)
	})

	t.Run("agent without AcceptsRemoteConfig gets False with explanation", func(t *testing.T) {
		t.Parallel()

		a := agentmodel.NewAgent(uuid.New()) // default capabilities: none
		withAssignedConfig(a)

		changed := svc.recordAgentRemoteConfigCondition(a)
		assert.True(t, changed)

		cond := a.GetCondition(agentmodel.AgentConditionTypeRemoteConfigApplied)
		require.NotNil(t, cond)
		assert.Equal(t, agentmodel.AgentConditionStatusFalse, cond.Status)
		assert.Contains(t, cond.Message, "does not accept remote config")
	})

	t.Run("no assigned config leaves the condition untouched", func(t *testing.T) {
		t.Parallel()

		a := agentmodel.NewAgent(uuid.New())

		changed := svc.recordAgentRemoteConfigCondition(a)

		assert.False(t, changed)
		assert.Nil(t, a.GetCondition(agentmodel.AgentConditionTypeRemoteConfigApplied))
	})

	t.Run("unchanged condition reports no change on repeat", func(t *testing.T) {
		t.Parallel()

		a := agentmodel.NewAgent(uuid.New())
		a.Metadata.Capabilities = agent.Capabilities(agent.AgentCapabilityAcceptsRemoteConfig)
		withAssignedConfig(a)

		assert.True(t, svc.recordAgentRemoteConfigCondition(a))
		assert.False(t, svc.recordAgentRemoteConfigCondition(a))
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
		alwaysLeaderElector{},
		inmemorystore.NewAgentGroupChangeStore(ChangedAgentGroupBufferSize),
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
	queued, err := svc.changeStore.Next(ctx)
	require.NoError(t, err)
	assert.Equal(t, "default", queued.Namespace)
	assert.Equal(t, "to-delete", queued.Name)
	assert.True(t, existing.IsDeleted())

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
		alwaysLeaderElector{},
		inmemorystore.NewAgentGroupChangeStore(ChangedAgentGroupBufferSize),
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
		svc := NewAgentGroupService(
			mockPersistence,
			mockRemoteConfigPort,
			mockCertPort,
			mockAgentUC,
			alwaysLeaderElector{},
			inmemorystore.NewAgentGroupChangeStore(ChangedAgentGroupBufferSize),
			logger,
		)

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
				Namespace: "default",
				Name:      "production",
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
		mockPersistence.On("ListAgentGroups", mock.Anything, mock.Anything, (*model.ListOptions)(nil)).
			Return(&model.ListResponse[*agentmodel.AgentGroup]{Items: []*agentmodel.AgentGroup{agentGroup}}, nil)
		mockRemoteConfigPort.On("GetAgentRemoteConfig", mock.Anything, "default", refName, (*model.GetOptions)(nil)).
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
		svc := NewAgentGroupService(
			mockPersistence,
			mockRemoteConfigPort,
			mockCertPort,
			mockAgentUC,
			alwaysLeaderElector{},
			inmemorystore.NewAgentGroupChangeStore(ChangedAgentGroupBufferSize),
			logger,
		)

		testAgent := agentmodel.NewAgent(uuid.New(), agentmodel.WithDescription(&agent.Description{
			IdentifyingAttributes: map[string]string{"service.name": "my-service"},
		}))

		inlineName := "inline-config"
		agentGroup := &agentmodel.AgentGroup{
			Metadata: agentmodel.AgentGroupMetadata{
				Namespace: "default",
				Name:      "staging",
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
		mockPersistence.On("ListAgentGroups", mock.Anything, mock.Anything, (*model.ListOptions)(nil)).
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

func TestReconcileAllAgents(t *testing.T) {
	t.Parallel()

	newSvc := func() (*AgentGroupService, *mockAgentGroupPersistence, *mockAgentUsecase) {
		mockPersistence := new(mockAgentGroupPersistence)
		mockAgentUC := new(mockAgentUsecase)
		svc := NewAgentGroupService(
			mockPersistence,
			new(mockRemoteConfigPersistence),
			new(mockCertPersistence),
			mockAgentUC,
			alwaysLeaderElector{},
			inmemorystore.NewAgentGroupChangeStore(ChangedAgentGroupBufferSize),
			slog.Default(),
		)

		return svc, mockPersistence, mockAgentUC
	}

	// agentWithStaleConfig is an agent that still carries a group's remote config in its spec.
	agentWithStaleConfig := func() *agentmodel.Agent {
		a := agentmodel.NewAgent(uuid.New(), agentmodel.WithDescription(&agent.Description{
			IdentifyingAttributes: map[string]string{"service.name": "changed"},
		}))
		a.Spec.RemoteConfig = &agentmodel.AgentSpecRemoteConfig{
			ConfigMap: agentmodel.AgentConfigMap{
				ConfigMap: map[string]agentmodel.AgentConfigFile{"prod/cfg": {Body: []byte("stale")}},
			},
		}

		return a
	}

	t.Run("clears config from an agent no longer selected by any group (identity-change orphan)", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		svc, mockPersistence, mockAgentUC := newSvc()

		orphan := agentWithStaleConfig()

		// A group still exists but its selector no longer matches the agent (its identity changed).
		group := &agentmodel.AgentGroup{
			Metadata: agentmodel.AgentGroupMetadata{Namespace: "default", Name: "prod"},
			Spec: agentmodel.AgentGroupSpec{
				Selector: agentmodel.AgentSelector{
					IdentifyingAttributes: map[string]string{"service.name": "was-matching"},
				},
			},
		}

		mockAgentUC.On("ListAgentsBySelector", ctx, agentmodel.AgentSelector{}, mock.Anything).
			Return(&model.ListResponse[*agentmodel.Agent]{Items: []*agentmodel.Agent{orphan}}, nil)
		mockPersistence.On("ListAgentGroups", mock.Anything, mock.Anything, (*model.ListOptions)(nil)).
			Return(&model.ListResponse[*agentmodel.AgentGroup]{Items: []*agentmodel.AgentGroup{group}}, nil)
		// The agent drifted (config -> none), so it must be persisted with the config dropped.
		mockAgentUC.On("SaveAgent", ctx, mock.MatchedBy(func(a *agentmodel.Agent) bool {
			return a.Spec.RemoteConfig == nil
		})).Return(nil)

		svc.reconcileAllAgents(ctx)

		mockAgentUC.AssertExpectations(t)
	})

	t.Run("does not re-save an agent whose desired spec is unchanged", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		svc, mockPersistence, mockAgentUC := newSvc()

		// Agent with no config and no matching group -> desired state already matches -> no write.
		a := agentmodel.NewAgent(uuid.New(), agentmodel.WithDescription(&agent.Description{
			IdentifyingAttributes: map[string]string{"service.name": "changed"},
		}))

		mockAgentUC.On("ListAgentsBySelector", ctx, agentmodel.AgentSelector{}, mock.Anything).
			Return(&model.ListResponse[*agentmodel.Agent]{Items: []*agentmodel.Agent{a}}, nil)
		mockPersistence.On("ListAgentGroups", mock.Anything, mock.Anything, (*model.ListOptions)(nil)).
			Return(&model.ListResponse[*agentmodel.AgentGroup]{Items: nil}, nil)

		svc.reconcileAllAgents(ctx)

		mockAgentUC.AssertNotCalled(t, "SaveAgent", mock.Anything, mock.Anything)
	})
}

// alwaysLeaderElector is a test double that always reports leadership, so the
// reconcile loop behaves as it did before leader election was introduced.
type alwaysLeaderElector struct{}

func (alwaysLeaderElector) IsLeader(context.Context) (bool, error) { return true, nil }

// fakeLeaderElector returns a configurable leadership result for reconcile-loop tests.
type fakeLeaderElector struct {
	leader bool
	err    error
}

func (f fakeLeaderElector) IsLeader(context.Context) (bool, error) { return f.leader, f.err }

var errBoomLeader = errors.New("leader election boom")

func TestAgentGroupService_reconcileAllIfLeader(t *testing.T) {
	t.Parallel()

	// The reconcile scan is cluster-wide by nature, so it asks for that explicitly
	// — ListAllAgentGroups rather than a namespaced listing with an empty name.
	listOpts := (*model.ListOptions)(nil)

	// noAgents lets the agent-centric reconcile pass (reconcileAllAgents) run to a clean
	// no-op: it lists agents once with an empty selector and gets nothing back.
	noAgents := func(m *mockAgentUsecase) *mockAgentUsecase {
		m.On("ListAgentsBySelector", mock.Anything, agentmodel.AgentSelector{}, mock.Anything).
			Return(&model.ListResponse[*agentmodel.Agent]{Items: nil}, nil)

		return m
	}

	t.Run("skips the reconcile scan when not leader", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		mockPersistence := new(mockAgentGroupPersistence)
		mockAgentUC := new(mockAgentUsecase)
		svc := NewAgentGroupService(
			mockPersistence,
			new(mockRemoteConfigPersistence),
			new(mockCertPersistence),
			mockAgentUC,
			fakeLeaderElector{leader: false, err: nil},
			inmemorystore.NewAgentGroupChangeStore(ChangedAgentGroupBufferSize),
			slog.Default(),
		)

		svc.reconcileAllIfLeader(ctx)

		mockPersistence.AssertNotCalled(t, "ListAgentGroups", mock.Anything, mock.Anything)
		mockAgentUC.AssertNotCalled(t, "ListAgentsBySelector", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("runs the reconcile scan when leader", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		mockPersistence := new(mockAgentGroupPersistence)
		mockPersistence.On("ListAllAgentGroups", mock.Anything, listOpts).
			Return(&model.ListResponse[*agentmodel.AgentGroup]{Items: nil}, nil)

		mockAgentUC := noAgents(new(mockAgentUsecase))
		svc := NewAgentGroupService(
			mockPersistence,
			new(mockRemoteConfigPersistence),
			new(mockCertPersistence),
			mockAgentUC,
			fakeLeaderElector{leader: true, err: nil},
			inmemorystore.NewAgentGroupChangeStore(ChangedAgentGroupBufferSize),
			slog.Default(),
		)

		svc.reconcileAllIfLeader(ctx)

		mockPersistence.AssertCalled(t, "ListAllAgentGroups", mock.Anything, listOpts)
		mockAgentUC.AssertCalled(t, "ListAgentsBySelector", mock.Anything, agentmodel.AgentSelector{}, mock.Anything)
	})

	t.Run("fails open and reconciles when leader election errors", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		mockPersistence := new(mockAgentGroupPersistence)
		mockPersistence.On("ListAllAgentGroups", mock.Anything, listOpts).
			Return(&model.ListResponse[*agentmodel.AgentGroup]{Items: nil}, nil)

		mockAgentUC := noAgents(new(mockAgentUsecase))
		svc := NewAgentGroupService(
			mockPersistence,
			new(mockRemoteConfigPersistence),
			new(mockCertPersistence),
			mockAgentUC,
			fakeLeaderElector{leader: false, err: errBoomLeader},
			inmemorystore.NewAgentGroupChangeStore(ChangedAgentGroupBufferSize),
			slog.Default(),
		)

		svc.reconcileAllIfLeader(ctx)

		mockPersistence.AssertCalled(t, "ListAllAgentGroups", mock.Anything, listOpts)
		mockAgentUC.AssertCalled(t, "ListAgentsBySelector", mock.Anything, agentmodel.AgentSelector{}, mock.Anything)
	})
}

func TestAgentGroupService_RunRetainsQueuedSelector(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()

		store := inmemorystore.NewAgentGroupChangeStore(ChangedAgentGroupBufferSize)
		queuedSelector := agentmodel.AgentSelector{IdentifyingAttributes: map[string]string{"service.name": "original"}}
		require.NoError(t, store.Enqueue(ctx, agentport.AgentGroupChange{
			Namespace: "default", Name: "group", Selector: queuedSelector,
		}))
		// Reload the latest configuration, including a group deleted after enqueue,
		// but still visit the agents selected when the event was queued.
		group := &agentmodel.AgentGroup{
			Metadata: agentmodel.AgentGroupMetadata{Namespace: "default", Name: "group", DeletedAt: time.Now()},
			Spec: agentmodel.AgentGroupSpec{Selector: agentmodel.AgentSelector{
				IdentifyingAttributes: map[string]string{"service.name": "latest"},
			}},
		}
		persistence := new(mockAgentGroupPersistence)
		persistence.On("GetAgentGroup", mock.Anything, "default", "group", &model.GetOptions{IncludeDeleted: true}).
			Return(group, nil).Once()

		agents := new(mockAgentUsecase)
		agents.On("ListAgentsBySelector", mock.Anything, queuedSelector, mock.Anything).
			Run(func(mock.Arguments) { cancel() }).Return(&model.ListResponse[*agentmodel.Agent]{Items: nil}, nil).Once()
		service := NewAgentGroupService(persistence, new(mockRemoteConfigPersistence), new(mockCertPersistence),
			agents, fakeLeaderElector{leader: false, err: nil}, store, slog.Default())
		require.NoError(t, service.Run(ctx))
		persistence.AssertExpectations(t)
		agents.AssertExpectations(t)
	})
}

func TestAgentGroupService_RunDrainsFormerMembersAfterRecreation(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()

		persistence := inmemorypersistence.NewAgentGroupRepository(inmemorypersistence.NewAgentRepository())
		original := agentmodel.NewAgentGroup("default", "group", nil, time.Now(), "admin")
		original.Spec.Selector.IdentifyingAttributes = map[string]string{"service.name": "original"}
		_, err := persistence.PutAgentGroup(ctx, "default", "group", original)
		require.NoError(t, err)

		member := agentmodel.NewAgent(uuid.New(), agentmodel.WithDescription(&agent.Description{
			IdentifyingAttributes: map[string]string{"service.name": "original"},
		}))
		setAgentRemoteConfigs(member, map[string]agentmodel.AgentConfigFile{"old": {Body: []byte("old config")}})

		recreated := agentmodel.NewAgentGroup("default", "group", nil, time.Now(), "admin")
		recreated.Spec.Selector.IdentifyingAttributes = map[string]string{"service.name": "recreated"}

		agents := new(mockAgentUsecase)
		agents.On("ListAgentsBySelector", mock.Anything, original.Spec.Selector, mock.Anything).
			Return(&model.ListResponse[*agentmodel.Agent]{Items: []*agentmodel.Agent{member}}, nil).Once()
		agents.On("SaveAgent", mock.Anything, mock.MatchedBy(func(saved *agentmodel.Agent) bool {
			return saved.Metadata.InstanceUID == member.Metadata.InstanceUID && saved.Spec.RemoteConfig == nil
		})).Return(nil).Once()
		agents.On("ListAgentsBySelector", mock.Anything, recreated.Spec.Selector, mock.Anything).
			Run(func(mock.Arguments) { cancel() }).Return(&model.ListResponse[*agentmodel.Agent]{}, nil).Once()

		svc := NewAgentGroupService(persistence, new(mockRemoteConfigPersistence), new(mockCertPersistence), agents,
			fakeLeaderElector{leader: false, err: nil},
			inmemorystore.NewAgentGroupChangeStore(ChangedAgentGroupBufferSize), slog.Default())
		require.NoError(t, svc.DeleteAgentGroup(ctx, "default", "group", time.Now(), "admin"))
		_, err = svc.SaveAgentGroup(ctx, "default", "group", recreated)
		require.NoError(t, err)
		require.NoError(t, svc.Run(ctx))
		require.Nil(t, member.Spec.RemoteConfig)

		current, err := persistence.GetAgentGroup(t.Context(), "default", "group", nil)
		require.NoError(t, err)
		assert.Equal(t, recreated.Spec.Selector, current.Spec.Selector)
		assert.False(t, current.IsDeleted())
		agents.AssertExpectations(t)
	})
}

func TestApplyMatchingAgentGroupsPriorityAndPagination(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name      string
		priorityA int
		priorityB int
		winner    string
		reverse   bool
		paged     bool
	}{
		{name: "higher priority first", priorityA: 10, priorityB: -10, winner: "a"},
		{name: "reversed insertion", priorityA: 10, priorityB: -10, winner: "a", reverse: true},
		{name: "reversed priorities", priorityA: -10, priorityB: 10, winner: "a/b"},
		{name: "equal priority", winner: "a"},
		{name: "equal priority reversed", winner: "a", reverse: true},
		{name: "winner on next page", priorityA: 10, priorityB: -10, winner: "a", reverse: true, paged: true},
		{name: "equal priority across pages", winner: "a", reverse: true, paged: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			first := agentmodel.NewAgentGroup("default", "a", nil, time.Now(), "admin")
			second := agentmodel.NewAgentGroup("default", "a/b", nil, time.Now(), "admin")
			first.Spec.Priority = test.priorityA
			second.Spec.Priority = test.priorityB

			for _, group := range []*agentmodel.AgentGroup{first, second} {
				configName := "shared"
				if group == first {
					configName = "b/shared"
				}

				group.Spec.AgentRemoteConfigs = []agentmodel.AgentGroupAgentRemoteConfig{{
					AgentRemoteConfigName: &configName,
					AgentRemoteConfigSpec: &agentmodel.AgentRemoteConfigSpec{
						Value: []byte(group.Metadata.Name), ContentType: "text/yaml",
					},
				}}
				group.Spec.AgentConnectionConfig = &agentmodel.AgentGroupConnectionConfig{
					OpAMPConnection: &agentmodel.OpAMPConnectionSettings{
						DestinationEndpoint: "wss://example.test/" + group.Metadata.Name,
						Headers:             map[string][]string{"group": {group.Metadata.Name}},
					},
				}
			}
			// The connection bundle is atomic: the loser's metrics must not mix with the winner's logs.
			first.Spec.AgentConnectionConfig.OwnLogs = &agentmodel.TelemetryConnectionSettings{
				DestinationEndpoint: "https://logs.test",
			}
			second.Spec.AgentConnectionConfig.OwnMetrics = &agentmodel.TelemetryConnectionSettings{
				DestinationEndpoint: "https://metrics.test",
			}
			unique := "unique"
			second.Spec.AgentRemoteConfigs = append(second.Spec.AgentRemoteConfigs, agentmodel.AgentGroupAgentRemoteConfig{
				AgentRemoteConfigName: &unique,
				AgentRemoteConfigSpec: &agentmodel.AgentRemoteConfigSpec{Value: []byte("unique"), ContentType: "text/yaml"},
			})

			groups := []*agentmodel.AgentGroup{first, second}
			if test.reverse {
				groups[0], groups[1] = groups[1], groups[0]
			}

			persistence := new(mockAgentGroupPersistence)

			response := &model.ListResponse[*agentmodel.AgentGroup]{Items: groups}
			if test.paged {
				response = &model.ListResponse[*agentmodel.AgentGroup]{Items: groups[:1], Continue: "next", RemainingItemCount: 1}
				persistence.On("ListAgentGroups", mock.Anything, "default", &model.ListOptions{Continue: "next"}).
					Return(&model.ListResponse[*agentmodel.AgentGroup]{Items: groups[1:]}, nil)
			}

			persistence.On("ListAgentGroups", mock.Anything, "default", (*model.ListOptions)(nil)).Return(response, nil)

			svc := NewAgentGroupService(persistence, new(mockRemoteConfigPersistence), new(mockCertPersistence),
				new(mockAgentUsecase), alwaysLeaderElector{},
				inmemorystore.NewAgentGroupChangeStore(ChangedAgentGroupBufferSize), slog.Default())
			member := agentmodel.NewAgent(uuid.New())
			require.NoError(t, svc.ApplyMatchingAgentGroupsToAgent(t.Context(), member))
			assert.Equal(t, []byte(test.winner), member.Spec.RemoteConfig.ConfigMap.ConfigMap["a/b/shared"].Body)
			assert.Equal(t, []byte("unique"), member.Spec.RemoteConfig.ConfigMap.ConfigMap["a/b/unique"].Body)
			assert.Equal(t, "wss://example.test/"+test.winner, member.Spec.ConnectionInfo.OpAMP().DestinationEndpoint)
			assert.Equal(t, []string{test.winner}, member.Spec.ConnectionInfo.OpAMP().Headers["group"])

			if test.winner == "a" {
				assert.Nil(t, member.Spec.ConnectionInfo.OwnMetrics())
				require.NotNil(t, member.Spec.ConnectionInfo.OwnLogs())
			} else {
				assert.Nil(t, member.Spec.ConnectionInfo.OwnLogs())
				require.NotNil(t, member.Spec.ConnectionInfo.OwnMetrics())
			}

			before := member.Clone()
			require.NoError(t, svc.ApplyMatchingAgentGroupsToAgent(t.Context(), member))
			assert.Equal(t, before, member)
			persistence.AssertExpectations(t)
		})
	}
}

func TestApplyMatchingAgentGroupsRemovesOwnedSettings(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name   string
		change func(*agentmodel.AgentGroup, *agentmodel.Agent)
	}{
		{name: "removed settings", change: func(group *agentmodel.AgentGroup, _ *agentmodel.Agent) {
			group.Spec.AgentRemoteConfigs = nil
			group.Spec.AgentConnectionConfig = nil
		}},
		{name: "deleted last group", change: func(group *agentmodel.AgentGroup, _ *agentmodel.Agent) {
			group.MarkDeleted(time.Now(), "admin")
		}},
		{name: "identity stops matching", change: func(_ *agentmodel.AgentGroup, member *agentmodel.Agent) {
			member.Metadata.Description.IdentifyingAttributes["service.name"] = "other"
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			persistence := inmemorypersistence.NewAgentGroupRepository(inmemorypersistence.NewAgentRepository())
			group := agentmodel.NewAgentGroup("default", "group", nil, time.Now(), "admin")
			group.Spec.Selector.IdentifyingAttributes = map[string]string{"service.name": "selected"}
			configName := "config"
			group.Spec.AgentRemoteConfigs = []agentmodel.AgentGroupAgentRemoteConfig{{
				AgentRemoteConfigName: &configName,
				AgentRemoteConfigSpec: &agentmodel.AgentRemoteConfigSpec{Value: []byte("config"), ContentType: "text/yaml"},
			}}
			group.Spec.AgentConnectionConfig = &agentmodel.AgentGroupConnectionConfig{
				OpAMPConnection: &agentmodel.OpAMPConnectionSettings{DestinationEndpoint: "wss://group.test"},
			}
			_, err := persistence.PutAgentGroup(t.Context(), "default", "group", group)
			require.NoError(t, err)

			svc := NewAgentGroupService(persistence, new(mockRemoteConfigPersistence), new(mockCertPersistence),
				new(mockAgentUsecase), alwaysLeaderElector{},
				inmemorystore.NewAgentGroupChangeStore(ChangedAgentGroupBufferSize), slog.Default())

			member := agentmodel.NewAgent(uuid.New(), agentmodel.WithDescription(&agent.Description{
				IdentifyingAttributes: map[string]string{"service.name": "selected"},
			}))
			require.NoError(t, svc.ApplyMatchingAgentGroupsToAgent(t.Context(), member))
			assert.Equal(t, "wss://group.test", member.Spec.ConnectionInfo.OpAMP().DestinationEndpoint)

			changed := *group
			test.change(&changed, member)
			_, err = persistence.PutAgentGroup(t.Context(), "default", "group", &changed)
			require.NoError(t, err)
			require.NoError(t, svc.ApplyMatchingAgentGroupsToAgent(t.Context(), member))
			assert.Nil(t, member.Spec.RemoteConfig)
			assert.Nil(t, member.Spec.ConnectionInfo)

			before := member.Clone()
			require.NoError(t, svc.ApplyMatchingAgentGroupsToAgent(t.Context(), member))
			assert.Equal(t, before, member)
		})
	}
}

func TestApplyMatchingAgentGroupsClearsUnmatchedConnectionOffer(t *testing.T) {
	t.Parallel()

	persistence := new(mockAgentGroupPersistence)
	persistence.On("ListAgentGroups", mock.Anything, "default", (*model.ListOptions)(nil)).
		Return(&model.ListResponse[*agentmodel.AgentGroup]{}, nil)
	svc := NewAgentGroupService(persistence, new(mockRemoteConfigPersistence), new(mockCertPersistence),
		new(mockAgentUsecase), alwaysLeaderElector{},
		inmemorystore.NewAgentGroupChangeStore(ChangedAgentGroupBufferSize), slog.Default())
	member := agentmodel.NewAgent(uuid.New())

	var err error

	member.Spec.ConnectionInfo, err = agentmodel.NewConnectionInfo(&agentmodel.AgentOpAMPConnectionSettings{
		DestinationEndpoint: "wss://unmatched-group.test",
	}, nil, nil, nil, nil)
	require.NoError(t, err)
	require.NoError(t, svc.ApplyMatchingAgentGroupsToAgent(t.Context(), member))
	assert.Nil(t, member.Spec.ConnectionInfo)
}

func TestGroupReconciliationDoesNotSaveUnchangedComposition(t *testing.T) {
	t.Parallel()

	persistence := inmemorypersistence.NewAgentGroupRepository(inmemorypersistence.NewAgentRepository())
	first := agentmodel.NewAgentGroup("default", "first", nil, time.Now(), "admin")
	second := agentmodel.NewAgentGroup("default", "second", nil, time.Now(), "admin")

	for _, group := range []*agentmodel.AgentGroup{first, second} {
		name := "config"
		group.Spec.AgentRemoteConfigs = []agentmodel.AgentGroupAgentRemoteConfig{{
			AgentRemoteConfigName: &name,
			AgentRemoteConfigSpec: &agentmodel.AgentRemoteConfigSpec{Value: []byte("config"), ContentType: "text/yaml"},
		}}
		_, err := persistence.PutAgentGroup(t.Context(), "default", group.Metadata.Name, group)
		require.NoError(t, err)
	}

	member := agentmodel.NewAgent(uuid.New())
	agents := new(mockAgentUsecase)
	agents.On("ListAgentsBySelector", mock.Anything, mock.Anything, mock.Anything).
		Return(&model.ListResponse[*agentmodel.Agent]{Items: []*agentmodel.Agent{member}}, nil)
	agents.On("SaveAgent", mock.Anything, member).Return(nil).Once()
	svc := NewAgentGroupService(persistence, new(mockRemoteConfigPersistence), new(mockCertPersistence),
		agents, alwaysLeaderElector{},
		inmemorystore.NewAgentGroupChangeStore(ChangedAgentGroupBufferSize), slog.Default())

	for range 2 {
		require.NoError(t, svc.updateAgentsByAgentGroup(t.Context(), first))
		require.NoError(t, svc.updateAgentsByAgentGroup(t.Context(), second))
	}

	agents.AssertNumberOfCalls(t, "SaveAgent", 1)
}

func TestApplyMatchingAgentGroupsReplacesConnectionFields(t *testing.T) {
	t.Parallel()

	persistence := inmemorypersistence.NewAgentGroupRepository(inmemorypersistence.NewAgentRepository())
	group := agentmodel.NewAgentGroup("default", "group", nil, time.Now(), "admin")
	group.Spec.AgentConnectionConfig = &agentmodel.AgentGroupConnectionConfig{
		OpAMPConnection: &agentmodel.OpAMPConnectionSettings{DestinationEndpoint: "wss://group.test"},
		OwnLogs:         &agentmodel.TelemetryConnectionSettings{DestinationEndpoint: "https://logs.test"},
		OtherConnections: map[string]agentmodel.OtherConnectionSettings{
			"other": {DestinationEndpoint: "https://other.test"},
		},
	}
	_, err := persistence.PutAgentGroup(t.Context(), "default", "group", group)
	require.NoError(t, err)

	svc := NewAgentGroupService(persistence, new(mockRemoteConfigPersistence), new(mockCertPersistence),
		new(mockAgentUsecase), alwaysLeaderElector{},
		inmemorystore.NewAgentGroupChangeStore(ChangedAgentGroupBufferSize), slog.Default())
	member := agentmodel.NewAgent(uuid.New())
	require.NoError(t, svc.ApplyMatchingAgentGroupsToAgent(t.Context(), member))
	require.NotNil(t, member.Spec.ConnectionInfo.OwnLogs())
	require.Contains(t, member.Spec.ConnectionInfo.OtherConnections(), "other")

	group.Spec.AgentConnectionConfig.OwnLogs = nil
	delete(group.Spec.AgentConnectionConfig.OtherConnections, "other")
	_, err = persistence.PutAgentGroup(t.Context(), "default", "group", group)
	require.NoError(t, err)
	require.NoError(t, svc.ApplyMatchingAgentGroupsToAgent(t.Context(), member))
	assert.Nil(t, member.Spec.ConnectionInfo.OwnLogs())
	assert.Empty(t, member.Spec.ConnectionInfo.OtherConnections())
	assert.Equal(t, "wss://group.test", member.Spec.ConnectionInfo.OpAMP().DestinationEndpoint)
}

func TestApplyMatchingAgentGroupsPageFailureLeavesAgentUnchanged(t *testing.T) {
	t.Parallel()

	persistence := new(mockAgentGroupPersistence)
	group := agentmodel.NewAgentGroup("default", "group", nil, time.Now(), "admin")
	persistence.On("ListAgentGroups", mock.Anything, "default", (*model.ListOptions)(nil)).
		Return(&model.ListResponse[*agentmodel.AgentGroup]{Items: []*agentmodel.AgentGroup{group}, Continue: "next"}, nil)
	persistence.On("ListAgentGroups", mock.Anything, "default", &model.ListOptions{Continue: "next"}).
		Return(nil, errRemoteConfigNotFound)
	svc := NewAgentGroupService(persistence, new(mockRemoteConfigPersistence), new(mockCertPersistence),
		new(mockAgentUsecase), alwaysLeaderElector{},
		inmemorystore.NewAgentGroupChangeStore(ChangedAgentGroupBufferSize), slog.Default())
	member := agentmodel.NewAgent(uuid.New())
	require.NoError(t, member.ApplyRemoteConfig("keep", agentmodel.AgentConfigFile{Body: []byte("keep")}))
	require.NoError(t, member.ApplyConnectionSettings(&agentmodel.AgentOpAMPConnectionSettings{
		DestinationEndpoint: "wss://keep.test",
	}, nil, nil, nil, nil))
	before := member.Clone()
	require.ErrorIs(t, svc.ApplyMatchingAgentGroupsToAgent(t.Context(), member), errRemoteConfigNotFound)
	assert.Equal(t, before, member)
}

func TestApplyMatchingAgentGroupsSelectsDeclaredConnectionBundle(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name   string
		config *agentmodel.AgentGroupConnectionConfig
	}{
		{name: "absent declaration uses lower-ranked supplier"},
		{name: "empty declaration overrides lower-ranked supplier", config: &agentmodel.AgentGroupConnectionConfig{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			higher := agentmodel.NewAgentGroup("default", "higher", nil, time.Now(), "admin")
			higher.Spec.Priority = 10
			higher.Spec.AgentConnectionConfig = test.config
			lower := agentmodel.NewAgentGroup("default", "lower", nil, time.Now(), "admin")
			lower.Spec.AgentConnectionConfig = &agentmodel.AgentGroupConnectionConfig{
				OpAMPConnection: &agentmodel.OpAMPConnectionSettings{DestinationEndpoint: "wss://lower.test"},
			}
			persistence := new(mockAgentGroupPersistence)
			persistence.On("ListAgentGroups", mock.Anything, "default", (*model.ListOptions)(nil)).
				Return(&model.ListResponse[*agentmodel.AgentGroup]{Items: []*agentmodel.AgentGroup{lower, higher}}, nil)
			svc := NewAgentGroupService(persistence, new(mockRemoteConfigPersistence), new(mockCertPersistence),
				new(mockAgentUsecase), alwaysLeaderElector{},
				inmemorystore.NewAgentGroupChangeStore(ChangedAgentGroupBufferSize), slog.Default())
			member := agentmodel.NewAgent(uuid.New())
			require.NoError(t, svc.ApplyMatchingAgentGroupsToAgent(t.Context(), member))
			require.NotNil(t, member.Spec.ConnectionInfo)

			if test.config == nil {
				assert.Equal(t, "wss://lower.test", member.Spec.ConnectionInfo.OpAMP().DestinationEndpoint)
			} else {
				assert.False(t, member.Spec.ConnectionInfo.HasConnectionSettings())
			}
		})
	}
}

func TestBuildOtherConnectionsOmitsUnresolvedCertificates(t *testing.T) {
	t.Parallel()

	certs := new(mockCertPersistence)
	certs.On("GetCertificate", mock.Anything, "default", "valid", (*model.GetOptions)(nil)).
		Return(&agentmodel.Certificate{
			Spec: agentmodel.CertificateSpec{Cert: []byte("cert"), PrivateKey: []byte("key")},
		}, nil)
	certs.On("GetCertificate", mock.Anything, "default", "missing", (*model.GetOptions)(nil)).
		Return(nil, model.ErrResourceNotExist)
	svc := NewAgentGroupService(new(mockAgentGroupPersistence), new(mockRemoteConfigPersistence), certs,
		new(mockAgentUsecase), alwaysLeaderElector{},
		inmemorystore.NewAgentGroupChangeStore(ChangedAgentGroupBufferSize), slog.Default())
	valid, missing := "valid", "missing"
	connections := svc.buildOtherConnections(t.Context(), "default", map[string]agentmodel.OtherConnectionSettings{
		"plain":     {DestinationEndpoint: "https://plain.test", Headers: map[string][]string{"token": {"value"}}},
		"certified": {DestinationEndpoint: "https://certified.test", CertificateName: &valid},
		"missing":   {DestinationEndpoint: "https://missing.test", CertificateName: &missing},
	}, slog.Default())
	assert.Len(t, connections, 2)
	assert.Equal(t, "https://plain.test", connections["plain"].DestinationEndpoint)
	assert.Equal(t, []string{"value"}, connections["plain"].Headers["token"])
	require.NotNil(t, connections["certified"].Certificate)
	assert.Equal(t, []byte("cert"), connections["certified"].Certificate.Cert)
	assert.Equal(t, []byte("key"), connections["certified"].Certificate.PrivateKey)
	assert.NotContains(t, connections, "missing")
	certs.AssertExpectations(t)
}
