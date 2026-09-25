package testutil

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/client"
	"github.com/open-telemetry/opamp-go/client/types"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/require"
)

const (
	referenceAgentHeartbeat   = 1 * time.Second
	referenceAgentStopTimeout = 5 * time.Second
)

// ReferenceAgentTransport selects the OpAMP transport a ReferenceAgent speaks.
type ReferenceAgentTransport string

const (
	// ReferenceAgentTransportWebSocket connects over the OpAMP WebSocket transport.
	ReferenceAgentTransportWebSocket ReferenceAgentTransport = "websocket"
	// ReferenceAgentTransportHTTP polls over the OpAMP plain-HTTP transport.
	ReferenceAgentTransportHTTP ReferenceAgentTransport = "http"
)

// ReferenceAgentCapabilities is what a ReferenceAgent advertises: every capability
// the reference behaviour below implements.
const ReferenceAgentCapabilities = protobufs.AgentCapabilities_AgentCapabilities_ReportsStatus |
	protobufs.AgentCapabilities_AgentCapabilities_AcceptsRemoteConfig |
	protobufs.AgentCapabilities_AgentCapabilities_ReportsEffectiveConfig |
	protobufs.AgentCapabilities_AgentCapabilities_ReportsRemoteConfig |
	protobufs.AgentCapabilities_AgentCapabilities_AcceptsPackages |
	protobufs.AgentCapabilities_AgentCapabilities_ReportsPackageStatuses |
	protobufs.AgentCapabilities_AgentCapabilities_ReportsOwnMetrics |
	protobufs.AgentCapabilities_AgentCapabilities_ReportsConnectionSettingsStatus |
	protobufs.AgentCapabilities_AgentCapabilities_AcceptsRestartCommand |
	protobufs.AgentCapabilities_AgentCapabilities_ReportsHealth |
	protobufs.AgentCapabilities_AgentCapabilities_ReportsHeartbeat

// ReferenceAgentOption customizes a ReferenceAgent.
type ReferenceAgentOption func(*referenceAgentSettings)

type referenceAgentSettings struct {
	transport      ReferenceAgentTransport
	identifying    map[string]string
	nonIdentifying map[string]string
}

// WithReferenceAgentTransport selects the OpAMP transport (WebSocket by default).
func WithReferenceAgentTransport(transport ReferenceAgentTransport) ReferenceAgentOption {
	return func(s *referenceAgentSettings) { s.transport = transport }
}

// WithReferenceAgentIdentifyingAttributes sets the identifying attributes reported
// in the AgentDescription, merged over the defaults.
func WithReferenceAgentIdentifyingAttributes(attrs map[string]string) ReferenceAgentOption {
	return func(s *referenceAgentSettings) { maps.Copy(s.identifying, attrs) }
}

// WithReferenceAgentNonIdentifyingAttributes sets the non-identifying attributes
// reported in the AgentDescription, merged over the defaults.
func WithReferenceAgentNonIdentifyingAttributes(attrs map[string]string) ReferenceAgentOption {
	return func(s *referenceAgentSettings) { maps.Copy(s.nonIdentifying, attrs) }
}

func newReferenceAgentSettings(opts []ReferenceAgentOption) referenceAgentSettings {
	settings := referenceAgentSettings{
		transport: ReferenceAgentTransportWebSocket,
		identifying: map[string]string{
			"service.name":    "opamp-reference-agent",
			"service.version": "0.0.0-test",
		},
		nonIdentifying: map[string]string{
			"os.type": "linux",
		},
	}
	for _, opt := range opts {
		opt(&settings)
	}

	return settings
}

// ReferenceAgent is an in-process OpAMP agent built on the upstream opamp-go client.
//
// It speaks the real wire protocol to the server and behaves as a well-behaved agent:
// it applies an offered remote config and reports it back as its effective config,
// acknowledges connection-settings offers, and "restarts" (bumps its health start
// time) on a Restart command. Unlike a collector container it exposes what it
// received, so a test can assert both sides of each round-trip.
type ReferenceAgent struct {
	UID       uuid.UUID
	Transport ReferenceAgentTransport

	client   client.OpAMPClient
	packages *memPackagesStore

	mu               sync.Mutex
	effectiveConfig  map[string]*protobufs.AgentConfigObject
	remoteConfigHash []byte
	remoteConfigs    int
	ownMetrics       *protobufs.TelemetryConnectionSettings
	restarts         int
	startTime        time.Time
	serverErrors     []*protobufs.ServerErrorResponse
}

// StartReferenceAgent connects a ReferenceAgent to the OpAMP endpoint of the server
// listening on opampPort. The agent is stopped when the test ends.
func (b *Base) StartReferenceAgent(opampPort int, opts ...ReferenceAgentOption) *ReferenceAgent {
	b.t.Helper()

	settings := newReferenceAgentSettings(opts)

	agent := &ReferenceAgent{
		UID:              uuid.New(),
		Transport:        settings.transport,
		client:           nil,
		packages:         newMemPackagesStore(),
		mu:               sync.Mutex{},
		effectiveConfig:  map[string]*protobufs.AgentConfigObject{},
		remoteConfigHash: nil,
		remoteConfigs:    0,
		ownMetrics:       nil,
		restarts:         0,
		startTime:        time.Now(),
		serverErrors:     nil,
	}

	var serverURL string

	agent.client, serverURL = newOpAMPClient(settings.transport, opampPort)

	require.NoError(b.t, agent.client.SetAgentDescription(&protobufs.AgentDescription{
		IdentifyingAttributes:    toKeyValues(settings.identifying),
		NonIdentifyingAttributes: toKeyValues(settings.nonIdentifying),
	}))

	capabilities := ReferenceAgentCapabilities
	require.NoError(b.t, agent.client.SetCapabilities(&capabilities))
	require.NoError(b.t, agent.client.SetHealth(agent.health(true, "StatusOK")))

	heartbeat := referenceAgentHeartbeat

	//exhaustruct:ignore
	err := agent.client.Start(b.t.Context(), types.StartSettings{
		OpAMPServerURL:        serverURL,
		InstanceUid:           types.InstanceUid(agent.UID),
		PackagesStateProvider: agent.packages,
		HeartbeatInterval:     &heartbeat,
		//exhaustruct:ignore
		Callbacks: types.Callbacks{
			OnMessage:          agent.onMessage,
			OnCommand:          agent.onCommand,
			OnError:            agent.onError,
			GetEffectiveConfig: agent.getEffectiveConfig,
		},
	})
	require.NoError(b.t, err, "reference agent should start")

	b.t.Cleanup(agent.Stop)

	return agent
}

// Stop disconnects the agent. It is safe to call more than once.
func (a *ReferenceAgent) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), referenceAgentStopTimeout)
	defer cancel()

	_ = a.client.Stop(ctx)
}

// EffectiveConfig returns the agent's current effective config as name → body.
func (a *ReferenceAgent) EffectiveConfig() map[string]string {
	a.mu.Lock()
	defer a.mu.Unlock()

	out := make(map[string]string, len(a.effectiveConfig))
	for name, file := range a.effectiveConfig {
		out[name] = string(file.GetBody())
	}

	return out
}

// RemoteConfigsApplied returns how many distinct remote configs the agent applied.
func (a *ReferenceAgent) RemoteConfigsApplied() int {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.remoteConfigs
}

// OwnMetricsEndpoint returns the own-metrics destination last offered by the server.
func (a *ReferenceAgent) OwnMetricsEndpoint() string {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.ownMetrics.GetDestinationEndpoint()
}

// Restarts returns how many Restart commands the agent executed.
func (a *ReferenceAgent) Restarts() int {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.restarts
}

// ServerErrors returns the error responses the server sent the agent.
func (a *ReferenceAgent) ServerErrors() []*protobufs.ServerErrorResponse {
	a.mu.Lock()
	defer a.mu.Unlock()

	return append([]*protobufs.ServerErrorResponse(nil), a.serverErrors...)
}

// ReportHealth reports the agent's top-level health and status string.
func (a *ReferenceAgent) ReportHealth(healthy bool, status string) error {
	//nolint:wrapcheck // test helper surfaces the client error as-is.
	return a.client.SetHealth(a.health(healthy, status))
}

// ReportPackageStatuses reports the given installed packages (name → version).
func (a *ReferenceAgent) ReportPackageStatuses(installed map[string]string) error {
	statuses := make(map[string]*protobufs.PackageStatus, len(installed))
	for name, version := range installed {
		//exhaustruct:ignore
		statuses[name] = &protobufs.PackageStatus{
			Name:            name,
			AgentHasVersion: version,
			Status:          protobufs.PackageStatusEnum_PackageStatusEnum_Installed,
		}
	}

	//nolint:wrapcheck // test helper surfaces the client error as-is.
	return a.client.SetPackageStatuses(&protobufs.PackageStatuses{
		Packages: statuses,
		// opamp-go refuses a report without it; nothing was offered, so any value will do.
		ServerProvidedAllPackagesHash: []byte("none"),
		ErrorMessage:                  "",
	})
}

func (a *ReferenceAgent) health(healthy bool, status string) *protobufs.ComponentHealth {
	a.mu.Lock()
	startTime := a.startTime
	a.mu.Unlock()

	now := uint64(time.Now().UnixNano())
	start := uint64(startTime.UnixNano())

	//exhaustruct:ignore
	return &protobufs.ComponentHealth{
		Healthy:            healthy,
		StartTimeUnixNano:  start,
		Status:             status,
		StatusTimeUnixNano: now,
		ComponentHealthMap: map[string]*protobufs.ComponentHealth{
			//exhaustruct:ignore
			"pipeline:traces": {
				Healthy:            healthy,
				StartTimeUnixNano:  start,
				Status:             status,
				StatusTimeUnixNano: now,
			},
		},
	}
}

func (a *ReferenceAgent) onMessage(ctx context.Context, msg *types.MessageData) {
	if msg.RemoteConfig != nil {
		a.applyRemoteConfig(ctx, msg.RemoteConfig)
	}

	if msg.OwnMetricsConnSettings != nil {
		a.mu.Lock()
		a.ownMetrics = msg.OwnMetricsConnSettings
		a.mu.Unlock()
	}
}

// applyRemoteConfig adopts the offered config verbatim as the effective config and
// acknowledges it, which is what a spec-compliant agent does on success. An offer whose
// hash matches the last applied one is unchanged and is ignored.
func (a *ReferenceAgent) applyRemoteConfig(ctx context.Context, remoteConfig *protobufs.AgentRemoteConfig) {
	a.mu.Lock()
	if bytes.Equal(a.remoteConfigHash, remoteConfig.GetConfigHash()) {
		a.mu.Unlock()

		return
	}

	a.remoteConfigHash = remoteConfig.GetConfigHash()
	a.effectiveConfig = maps.Clone(remoteConfig.GetConfig().GetConfigMap())
	a.remoteConfigs++
	a.mu.Unlock()

	_ = a.client.SetRemoteConfigStatus(&protobufs.RemoteConfigStatus{
		LastRemoteConfigHash: remoteConfig.GetConfigHash(),
		Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED,
		ErrorMessage:         "",
	})
	_ = a.client.UpdateEffectiveConfig(ctx)
}

func (a *ReferenceAgent) onCommand(_ context.Context, command *protobufs.ServerToAgentCommand) error {
	if command.GetType() != protobufs.CommandType_CommandType_Restart {
		return nil
	}

	a.mu.Lock()
	a.restarts++
	a.startTime = time.Now()
	a.mu.Unlock()

	// A restarted agent reports its new start time, which is how the server learns the
	// restart it asked for has happened.
	return a.ReportHealth(true, "StatusOK")
}

func (a *ReferenceAgent) onError(_ context.Context, errResponse *protobufs.ServerErrorResponse) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.serverErrors = append(a.serverErrors, errResponse)
}

func (a *ReferenceAgent) getEffectiveConfig(_ context.Context) (*protobufs.EffectiveConfig, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	return &protobufs.EffectiveConfig{
		ConfigMap: &protobufs.AgentConfigMap{ConfigMap: maps.Clone(a.effectiveConfig)},
	}, nil
}

// newOpAMPClient returns an unstarted opamp-go client for the transport and the
// server URL it should connect to.
//
//nolint:ireturn // opamp-go only exposes its clients through this interface.
func newOpAMPClient(transport ReferenceAgentTransport, opampPort int) (client.OpAMPClient, string) {
	if transport == ReferenceAgentTransportHTTP {
		return client.NewHTTP(nil), fmt.Sprintf("http://localhost:%d/api/v1/opamp", opampPort)
	}

	return client.NewWebSocket(nil), fmt.Sprintf("ws://localhost:%d/api/v1/opamp", opampPort)
}

func toKeyValues(attrs map[string]string) []*protobufs.KeyValue {
	out := make([]*protobufs.KeyValue, 0, len(attrs))
	for key, value := range attrs {
		out = append(out, &protobufs.KeyValue{
			Key: key,
			Value: &protobufs.AnyValue{
				Value: &protobufs.AnyValue_StringValue{StringValue: value},
			},
		})
	}

	return out
}

var errPackageExists = errors.New("package already exists")

// memPackagesStore is an in-memory types.PackagesStateProvider, enough to let the
// reference agent advertise AcceptsPackages / ReportsPackageStatuses.
type memPackagesStore struct {
	mu              sync.Mutex
	allPackagesHash []byte
	states          map[string]types.PackageState
	content         map[string][]byte
	lastStatuses    *protobufs.PackageStatuses
}

var _ types.PackagesStateProvider = (*memPackagesStore)(nil)

func newMemPackagesStore() *memPackagesStore {
	return &memPackagesStore{
		mu:              sync.Mutex{},
		allPackagesHash: nil,
		states:          map[string]types.PackageState{},
		content:         map[string][]byte{},
		lastStatuses:    nil,
	}
}

func (s *memPackagesStore) AllPackagesHash() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.allPackagesHash, nil
}

func (s *memPackagesStore) SetAllPackagesHash(hash []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.allPackagesHash = hash

	return nil
}

func (s *memPackagesStore) Packages() ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	names := make([]string, 0, len(s.states))
	for name := range s.states {
		names = append(names, name)
	}

	return names, nil
}

func (s *memPackagesStore) PackageState(packageName string) (types.PackageState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.states[packageName], nil
}

func (s *memPackagesStore) SetPackageState(packageName string, state types.PackageState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.states[packageName] = state

	return nil
}

func (s *memPackagesStore) CreatePackage(packageName string, typ protobufs.PackageType) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.states[packageName]; ok {
		return errPackageExists
	}

	s.states[packageName] = types.PackageState{Exists: true, Type: typ, Hash: nil, Version: ""}

	return nil
}

func (s *memPackagesStore) FileContentHash(packageName string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.content[packageName]; !ok {
		return nil, nil
	}

	return s.states[packageName].Hash, nil
}

func (s *memPackagesStore) UpdateContent(
	_ context.Context, packageName, _ string, data io.Reader, _, _ []byte,
) error {
	body, err := io.ReadAll(data)
	if err != nil {
		return fmt.Errorf("read package %s: %w", packageName, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.content[packageName] = body

	return nil
}

func (s *memPackagesStore) DeletePackage(packageName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.states, packageName)
	delete(s.content, packageName)

	return nil
}

func (s *memPackagesStore) LastReportedStatuses() (*protobufs.PackageStatuses, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.lastStatuses, nil
}

func (s *memPackagesStore) SetLastReportedStatuses(statuses *protobufs.PackageStatuses) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastStatuses = statuses

	return nil
}
