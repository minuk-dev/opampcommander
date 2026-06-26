package agentservice_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/port"
)

// fakeEndpointUsecase is an in-memory agentport.EndpointUsecase for tests, with
// soft-delete semantics matching the real persistence.
type fakeEndpointUsecase struct {
	store map[string]*agentmodel.Endpoint // key: namespace/name
}

func newFakeEndpointUsecase() *fakeEndpointUsecase {
	return &fakeEndpointUsecase{store: map[string]*agentmodel.Endpoint{}}
}

func key(namespace, name string) string { return namespace + "/" + name }

func (f *fakeEndpointUsecase) GetEndpoint(
	_ context.Context, namespace, name string, options *model.GetOptions,
) (*agentmodel.Endpoint, error) {
	endpoint, ok := f.store[key(namespace, name)]
	if !ok {
		return nil, port.ErrResourceNotExist
	}

	if endpoint.IsDeleted() && (options == nil || !options.IncludeDeleted) {
		return nil, port.ErrResourceNotExist
	}

	return endpoint, nil
}

func (f *fakeEndpointUsecase) ListEndpoints(
	_ context.Context, namespace string, _ *model.ListOptions,
) (*model.ListResponse[*agentmodel.Endpoint], error) {
	items := make([]*agentmodel.Endpoint, 0, len(f.store))

	for _, endpoint := range f.store {
		if endpoint.Metadata.Namespace == namespace && !endpoint.IsDeleted() {
			items = append(items, endpoint)
		}
	}

	return &model.ListResponse[*agentmodel.Endpoint]{Items: items, Continue: "", RemainingItemCount: 0}, nil
}

func (f *fakeEndpointUsecase) SaveEndpoint(
	_ context.Context, endpoint *agentmodel.Endpoint,
) (*agentmodel.Endpoint, error) {
	f.store[key(endpoint.Metadata.Namespace, endpoint.Metadata.Name)] = endpoint

	return endpoint, nil
}

func (f *fakeEndpointUsecase) CreateEndpoint(
	_ context.Context, endpoint *agentmodel.Endpoint, _ string,
) (*agentmodel.Endpoint, error) {
	f.store[key(endpoint.Metadata.Namespace, endpoint.Metadata.Name)] = endpoint

	return endpoint, nil
}

func (f *fakeEndpointUsecase) UpdateEndpoint(
	_ context.Context, namespace, name string, endpoint *agentmodel.Endpoint,
) (*agentmodel.Endpoint, error) {
	f.store[key(namespace, name)] = endpoint

	return endpoint, nil
}

func (f *fakeEndpointUsecase) DeleteEndpoint(
	_ context.Context, namespace, name string, deletedAt time.Time, deletedBy string,
) error {
	endpoint, ok := f.store[key(namespace, name)]
	if !ok {
		return port.ErrResourceNotExist
	}

	endpoint.MarkDeleted(deletedAt, deletedBy)

	return nil
}

const otlpAndMimirConfig = `
exporters:
  otlp:
    endpoint: https://otlp.example.com:4317
  prometheusremotewrite/mimir:
    endpoint: https://mimir.example.com/api/v1/push
    headers:
      X-Scope-OrgID: team-a
  debug:
    verbosity: basic
service:
  pipelines:
    traces:
      exporters: [otlp]
    metrics:
      exporters: [otlp, prometheusremotewrite/mimir]
`

func newRemoteConfig(config string) *agentmodel.AgentRemoteConfig {
	//exhaustruct:ignore
	return &agentmodel.AgentRemoteConfig{
		Metadata: agentmodel.AgentRemoteConfigMetadata{Namespace: "default", Name: "obs"},
		Spec:     agentmodel.AgentRemoteConfigSpec{Value: []byte(config), ContentType: "text/yaml"},
	}
}

func TestReconcileAutoCreatesEndpoints(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fake := newFakeEndpointUsecase()
	svc := agentservice.NewEndpointDetectionService(fake, slog.Default())

	require.NoError(t, svc.ReconcileEndpointsFromRemoteConfig(ctx, newRemoteConfig(otlpAndMimirConfig)))

	// debug has no endpoint/url, so only otlp + mimir become endpoints.
	list, err := fake.ListEndpoints(ctx, "default", nil)
	require.NoError(t, err)
	assert.Len(t, list.Items, 2)

	otlp, err := fake.GetEndpoint(ctx, "default", "obs-otlp", nil)
	require.NoError(t, err)
	assert.Equal(t, "https://otlp.example.com:4317", otlp.Spec.URL)
	assert.Equal(t, "otlp", otlp.Spec.Protocol)
	assert.True(t, otlp.Spec.Signals.Traces)
	assert.True(t, otlp.Spec.Signals.Metrics)
	assert.False(t, otlp.Spec.Signals.Logs)
	assert.Equal(t, "default/obs", otlp.Metadata.Attributes[agentservice.EndpointGeneratedFromAttribute])
	assert.Equal(t, "default/obs", otlp.Metadata.Attributes[agentservice.EndpointMatchedByAttribute])

	mimir, err := fake.GetEndpoint(ctx, "default", "obs-prometheusremotewrite-mimir", nil)
	require.NoError(t, err)
	assert.Equal(t, "prometheusremotewrite", mimir.Spec.Protocol)
	assert.True(t, mimir.Spec.Signals.Metrics)
	require.Len(t, mimir.Spec.Tenants, 1)
	assert.Equal(t, "team-a", mimir.Spec.Tenants[0].Name)
	assert.Equal(t, "team-a", mimir.Spec.Tenants[0].Headers["X-Scope-OrgID"])
}

func TestReconcileMatchesExistingByURLPreservingSpec(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fake := newFakeEndpointUsecase()
	svc := agentservice.NewEndpointDetectionService(fake, slog.Default())

	// A manually-created endpoint for the same URL the otlp exporter targets.
	manual := agentmodel.NewEndpoint("default", "my-collector", agentmodel.Attributes{}, time.Now(), "alice")
	manual.Spec.URL = "https://otlp.example.com:4317"
	manual.Spec.Protocol = "otlp"
	manual.Spec.Signals = agentmodel.EndpointSignals{Metrics: false, Logs: false, Traces: false}
	_, err := fake.SaveEndpoint(ctx, manual)
	require.NoError(t, err)

	require.NoError(t, svc.ReconcileEndpointsFromRemoteConfig(ctx, newRemoteConfig(otlpAndMimirConfig)))

	// The otlp exporter matches the manual endpoint by URL (no duplicate created);
	// only the mimir exporter is auto-created.
	_, err = fake.GetEndpoint(ctx, "default", "obs-otlp", &model.GetOptions{IncludeDeleted: true})
	require.ErrorIs(t, err, port.ErrResourceNotExist)

	matched, err := fake.GetEndpoint(ctx, "default", "my-collector", nil)
	require.NoError(t, err)
	// Linked to the remote config...
	assert.Equal(t, "default/obs", matched.Metadata.Attributes[agentservice.EndpointMatchedByAttribute])
	// ...but its spec is preserved (signals not overwritten by the config).
	assert.False(t, matched.Spec.Signals.Traces)
	assert.False(t, matched.Spec.Signals.Metrics)
	// not auto-created, so no generated-from marker.
	assert.Empty(t, matched.Metadata.Attributes[agentservice.EndpointGeneratedFromAttribute])

	list, err := fake.ListEndpoints(ctx, "default", nil)
	require.NoError(t, err)
	assert.Len(t, list.Items, 2) // my-collector (matched) + obs-prometheusremotewrite-mimir (created)
}

func TestReconcileNeverDeletes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fake := newFakeEndpointUsecase()
	svc := agentservice.NewEndpointDetectionService(fake, slog.Default())

	require.NoError(t, svc.ReconcileEndpointsFromRemoteConfig(ctx, newRemoteConfig(otlpAndMimirConfig)))

	// Re-reconcile with the mimir exporter removed: its endpoint must NOT be deleted.
	const otlpOnly = `
exporters:
  otlp:
    endpoint: https://otlp.example.com:4317
service:
  pipelines:
    traces:
      exporters: [otlp]
`
	require.NoError(t, svc.ReconcileEndpointsFromRemoteConfig(ctx, newRemoteConfig(otlpOnly)))

	list, err := fake.ListEndpoints(ctx, "default", nil)
	require.NoError(t, err)
	assert.Len(t, list.Items, 2, "removing an exporter must not delete its endpoint")
}

func TestReconcileCollapsesExportersSharingURL(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fake := newFakeEndpointUsecase()
	svc := agentservice.NewEndpointDetectionService(fake, slog.Default())

	// Two exporters point at the same URL but carry different signals; they must
	// collapse to a single endpoint whose signals are the union.
	const sharedURL = `
exporters:
  otlp/traces:
    endpoint: https://collector.example.com:4317
  otlphttp/metrics:
    endpoint: https://collector.example.com:4317
service:
  pipelines:
    traces:
      exporters: [otlp/traces]
    metrics:
      exporters: [otlphttp/metrics]
`
	require.NoError(t, svc.ReconcileEndpointsFromRemoteConfig(ctx, newRemoteConfig(sharedURL)))

	list, err := fake.ListEndpoints(ctx, "default", nil)
	require.NoError(t, err)
	require.Len(t, list.Items, 1, "exporters sharing a URL collapse to one endpoint")

	endpoint := list.Items[0]
	assert.Equal(t, "https://collector.example.com:4317", endpoint.Spec.URL)
	// Deterministic winner is the lowest sorted key ("otlp/traces"); signals merged.
	assert.Equal(t, "obs-otlp-traces", endpoint.Metadata.Name)
	assert.True(t, endpoint.Spec.Signals.Traces)
	assert.True(t, endpoint.Spec.Signals.Metrics)
}

func TestReconcileSkipsNonScalarHeaderValues(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fake := newFakeEndpointUsecase()
	svc := agentservice.NewEndpointDetectionService(fake, slog.Default())

	// A header whose value is a mapping must be skipped, not stringified to garbage.
	const badHeader = `
exporters:
  otlp:
    endpoint: https://otlp.example.com:4317
    headers:
      X-Scope-OrgID: team-a
      bad:
        nested: value
service:
  pipelines:
    metrics:
      exporters: [otlp]
`
	require.NoError(t, svc.ReconcileEndpointsFromRemoteConfig(ctx, newRemoteConfig(badHeader)))

	endpoint, err := fake.GetEndpoint(ctx, "default", "obs-otlp", nil)
	require.NoError(t, err)
	require.Len(t, endpoint.Spec.Tenants, 1)
	headers := endpoint.Spec.Tenants[0].Headers
	assert.Equal(t, "team-a", headers["X-Scope-OrgID"])
	_, hasBad := headers["bad"]
	assert.False(t, hasBad, "non-scalar header value must be skipped")
}

func TestExtractEndpointsFromAgent(t *testing.T) {
	t.Parallel()

	fake := newFakeEndpointUsecase()
	svc := agentservice.NewEndpointDetectionService(fake, slog.Default())

	agent := agentmodel.NewAgent(uuid.New())
	agent.Status.EffectiveConfig = agentmodel.AgentEffectiveConfig{
		ConfigMap: agentmodel.AgentConfigMap{
			ConfigMap: map[string]agentmodel.AgentConfigFile{
				"collector.yaml": {Body: []byte(otlpAndMimirConfig), ContentType: "text/yaml"},
			},
		},
	}

	endpoints, err := svc.ExtractEndpointsFromAgent(agent)
	require.NoError(t, err)
	require.Len(t, endpoints, 2)

	byName := map[string]*agentmodel.Endpoint{}
	for _, e := range endpoints {
		byName[e.Metadata.Name] = e
	}

	otlp := byName["otlp"]
	require.NotNil(t, otlp)
	assert.Equal(t, "https://otlp.example.com:4317", otlp.Spec.URL)
	assert.True(t, otlp.Spec.Signals.Traces)
	assert.True(t, otlp.Spec.Signals.Metrics)
	assert.Equal(t, "agent/"+agent.Metadata.InstanceUID.String(),
		otlp.Metadata.Attributes[agentservice.EndpointExtractedFromAttribute])

	mimir := byName["prometheusremotewrite-mimir"]
	require.NotNil(t, mimir)
	require.Len(t, mimir.Spec.Tenants, 1)
	assert.Equal(t, "team-a", mimir.Spec.Tenants[0].Name)

	// Extraction is read-only: nothing was persisted.
	list, err := fake.ListEndpoints(context.Background(), "default", nil)
	require.NoError(t, err)
	assert.Empty(t, list.Items)
}

func TestReconcileParseErrorKeepsExisting(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fake := newFakeEndpointUsecase()
	svc := agentservice.NewEndpointDetectionService(fake, slog.Default())

	require.NoError(t, svc.ReconcileEndpointsFromRemoteConfig(ctx, newRemoteConfig(otlpAndMimirConfig)))

	// A subsequent broken config must error without touching existing endpoints.
	err := svc.ReconcileEndpointsFromRemoteConfig(ctx, newRemoteConfig("\tnot: [valid"))
	require.Error(t, err)

	list, listErr := fake.ListEndpoints(ctx, "default", nil)
	require.NoError(t, listErr)
	assert.Len(t, list.Items, 2)
}
