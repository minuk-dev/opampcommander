package agentservice_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/inmemory"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
)

const sampleCollectorConfig = `
receivers:
  otlp:
    protocols:
      grpc:
  otlp/2:
processors:
  batch:
exporters:
  otlp/backend:
    endpoint: backend:4317
service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [otlp/backend]
`

func seedSchema(
	t *testing.T, svc *agentservice.RemoteConfigSchemaService, name string, components agentmodel.ComponentCatalog,
) {
	t.Helper()

	schema := agentmodel.NewRemoteConfigSchema("default", name, nil, time.Now(), "tester")
	schema.Spec.Components = components

	_, err := svc.CreateRemoteConfigSchema(t.Context(), schema, "tester")
	require.NoError(t, err)
}

func newSchemaRemoteConfig(content string) *agentmodel.AgentRemoteConfig {
	return &agentmodel.AgentRemoteConfig{
		Metadata: agentmodel.AgentRemoteConfigMetadata{Name: "cfg", Namespace: "default"},
		Spec:     agentmodel.AgentRemoteConfigSpec{Value: []byte(content), ContentType: "application/yaml"},
	}
}

func TestRemoteConfigSchemaService_ResolveSchemaRefs(t *testing.T) {
	t.Parallel()

	repo := inmemory.NewRemoteConfigSchemaRepository()
	svc := agentservice.NewRemoteConfigSchemaService(repo)
	ctx := t.Context()

	// Compatible: contains every used component (otlp receiver, batch processor, otlp exporter).
	seedSchema(t, svc, "contrib", agentmodel.ComponentCatalog{
		"receivers":  {"otlp", "hostmetrics"},
		"processors": {"batch", "memory_limiter"},
		"exporters":  {"otlp", "debug"},
	})
	// Incompatible: missing the batch processor.
	seedSchema(t, svc, "core", agentmodel.ComponentCatalog{
		"receivers": {"otlp"},
		"exporters": {"otlp"},
	})

	refs, err := svc.ResolveSchemaRefs(ctx, newSchemaRemoteConfig(sampleCollectorConfig))
	require.NoError(t, err)
	assert.Equal(t, []string{"contrib"}, refs)
}

func TestRemoteConfigSchemaService_ResolveSchemaRefs_UnparseableYieldsNone(t *testing.T) {
	t.Parallel()

	svc := agentservice.NewRemoteConfigSchemaService(inmemory.NewRemoteConfigSchemaRepository())

	refs, err := svc.ResolveSchemaRefs(t.Context(), newSchemaRemoteConfig("\tnot: [valid"))
	require.NoError(t, err)
	assert.Empty(t, refs)
}

// TestRemoteConfigSchemaService_ResolveSchemaRefs_NoComponentsMatchesNone verifies that a
// config declaring no components matches no schema (rather than matching every schema
// vacuously).
func TestRemoteConfigSchemaService_ResolveSchemaRefs_NoComponentsMatchesNone(t *testing.T) {
	t.Parallel()

	svc := agentservice.NewRemoteConfigSchemaService(inmemory.NewRemoteConfigSchemaRepository())
	seedSchema(t, svc, "contrib", agentmodel.ComponentCatalog{"receivers": {"otlp"}})

	refs, err := svc.ResolveSchemaRefs(t.Context(), newSchemaRemoteConfig("service:\n  pipelines: {}\n"))
	require.NoError(t, err)
	assert.Empty(t, refs)
}

// TestAgentRemoteConfigService_AutoResolvesSchemaRefs verifies that creating an
// AgentRemoteConfig without explicit SchemaRefs auto-populates them from the
// compatible schemas.
func TestAgentRemoteConfigService_AutoResolvesSchemaRefs(t *testing.T) {
	t.Parallel()

	schemaRepo := inmemory.NewRemoteConfigSchemaRepository()
	matcher := agentservice.NewRemoteConfigSchemaService(schemaRepo)
	seedSchema(t, matcher, "contrib", agentmodel.ComponentCatalog{
		"receivers":  {"otlp"},
		"processors": {"batch"},
		"exporters":  {"otlp"},
	})

	arcRepo := inmemory.NewAgentRemoteConfigRepository()
	arcSvc := agentservice.NewAgentRemoteConfigService(arcRepo, nil, nil, matcher, nil)

	config := newSchemaRemoteConfig(sampleCollectorConfig)

	created, err := arcSvc.CreateAgentRemoteConfig(t.Context(), config, "tester")
	require.NoError(t, err)
	assert.Equal(t, []string{"contrib"}, created.Spec.SchemaRefs)
}

// TestAgentRemoteConfigService_KeepsExplicitSchemaRefs verifies explicit SchemaRefs
// are preserved (auto-resolution only fills when empty).
func TestAgentRemoteConfigService_KeepsExplicitSchemaRefs(t *testing.T) {
	t.Parallel()

	schemaRepo := inmemory.NewRemoteConfigSchemaRepository()
	matcher := agentservice.NewRemoteConfigSchemaService(schemaRepo)
	seedSchema(t, matcher, "contrib", agentmodel.ComponentCatalog{"receivers": {"otlp"}})

	arcSvc := agentservice.NewAgentRemoteConfigService(
		inmemory.NewAgentRemoteConfigRepository(), nil, nil, matcher, nil)

	config := newSchemaRemoteConfig(sampleCollectorConfig)
	config.Spec.SchemaRefs = []string{"pinned"}

	created, err := arcSvc.CreateAgentRemoteConfig(t.Context(), config, "tester")
	require.NoError(t, err)
	assert.Equal(t, []string{"pinned"}, created.Spec.SchemaRefs)
}

// TestAgentRemoteConfigService_SkipAnnotationBypassesAutoResolve verifies the
// skip-schema-validation annotation prevents auto-resolution even when a matching schema
// exists.
func TestAgentRemoteConfigService_SkipAnnotationBypassesAutoResolve(t *testing.T) {
	t.Parallel()

	schemaRepo := inmemory.NewRemoteConfigSchemaRepository()
	matcher := agentservice.NewRemoteConfigSchemaService(schemaRepo)
	seedSchema(t, matcher, "contrib", agentmodel.ComponentCatalog{
		"receivers":  {"otlp"},
		"processors": {"batch"},
		"exporters":  {"otlp"},
	})

	arcSvc := agentservice.NewAgentRemoteConfigService(
		inmemory.NewAgentRemoteConfigRepository(), nil, nil, matcher, nil)

	config := newSchemaRemoteConfig(sampleCollectorConfig)
	config.Metadata.Attributes = agentmodel.Attributes{
		agentmodel.SkipSchemaValidationAnnotation: "true",
	}

	created, err := arcSvc.CreateAgentRemoteConfig(t.Context(), config, "tester")
	require.NoError(t, err)
	assert.Empty(t, created.Spec.SchemaRefs)
}

// TestAgentRemoteConfigService_UpdateDoesNotAutoResolve verifies auto-resolution runs only
// on create: an update can clear SchemaRefs without them being re-derived.
func TestAgentRemoteConfigService_UpdateDoesNotAutoResolve(t *testing.T) {
	t.Parallel()

	schemaRepo := inmemory.NewRemoteConfigSchemaRepository()
	matcher := agentservice.NewRemoteConfigSchemaService(schemaRepo)
	seedSchema(t, matcher, "contrib", agentmodel.ComponentCatalog{
		"receivers":  {"otlp"},
		"processors": {"batch"},
		"exporters":  {"otlp"},
	})

	arcSvc := agentservice.NewAgentRemoteConfigService(
		inmemory.NewAgentRemoteConfigRepository(), nil, nil, matcher, nil)
	ctx := t.Context()

	created, err := arcSvc.CreateAgentRemoteConfig(ctx, newSchemaRemoteConfig(sampleCollectorConfig), "tester")
	require.NoError(t, err)
	require.Equal(t, []string{"contrib"}, created.Spec.SchemaRefs)

	// Update with SchemaRefs cleared: the update must not re-derive them.
	update := newSchemaRemoteConfig(sampleCollectorConfig)
	update.Spec.SchemaRefs = nil

	updated, err := arcSvc.UpdateAgentRemoteConfig(ctx, "default", "cfg", update)
	require.NoError(t, err)
	assert.Empty(t, updated.Spec.SchemaRefs)
}
