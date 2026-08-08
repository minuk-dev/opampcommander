package infrastructure

import (
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

// schemaDir is the in-memory directory schema library files are written to.
const schemaDir = "/schemas"

const schemaOtelcol129 = `kind: RemoteConfigSchema
apiVersion: v1
metadata:
  name: otelcol-0.129.0
  namespace: default
spec:
  binary: otelcol
  version: 0.129.0
  components:
    receivers:
      otlp:
        type: otlp
`

const schemaOtelcol130 = `kind: RemoteConfigSchema
apiVersion: v1
metadata:
  name: otelcol-0.130.0
  namespace: default
spec:
  binary: otelcol
  version: 0.130.0
  components:
    receivers:
      otlp:
        type: otlp
        signals: [traces, metrics, logs]
      hostmetrics:
        type: hostmetrics
        signals: [metrics]
`

const schemaContrib130 = `kind: RemoteConfigSchema
apiVersion: v1
metadata:
  name: otelcol-contrib-0.130.0
  namespace: default
spec:
  binary: otelcol-contrib
  version: 0.130.0
  components:
    receivers:
      otlp:
        type: otlp
`

// writeSchemas writes the given schema library files into schemaDir on fs.
func writeSchemas(t *testing.T, fs afero.Fs, files map[string]string) string {
	t.Helper()

	for name, content := range files {
		require.NoError(t, afero.WriteFile(fs, filepath.Join(schemaDir, name), []byte(content), 0o600))
	}

	return schemaDir
}

func allThreeSchemas() map[string]string {
	return map[string]string{
		"otelcol-0.129.0.yaml":         schemaOtelcol129,
		"otelcol-0.130.0.yaml":         schemaOtelcol130,
		"otelcol-contrib-0.130.0.yaml": schemaContrib130,
	}
}

func schemaExists(t *testing.T, deps bootstrapDeps, name string) bool {
	t.Helper()

	_, err := deps.schemaUsecase.GetRemoteConfigSchema(t.Context(), "default", name, nil)
	if err == nil {
		return true
	}

	require.ErrorIs(t, err, model.ErrResourceNotExist)

	return false
}

func TestReconcileRemoteConfigSchemas_LatestSeedsNewestPerBinary(t *testing.T) {
	t.Parallel()

	deps, _, _ := newTestDeps()
	dir := writeSchemas(t, deps.fs, allThreeSchemas())

	require.NoError(t, reconcileRemoteConfigSchemas(t.Context(), dir, config.RemoteConfigSchemaLoadLatest, deps))

	assert.True(t, schemaExists(t, deps, "otelcol-0.130.0"), "newest otelcol seeded")
	assert.True(t, schemaExists(t, deps, "otelcol-contrib-0.130.0"), "newest contrib seeded")
	assert.False(t, schemaExists(t, deps, "otelcol-0.129.0"), "older otelcol not seeded under latest")
}

func TestReconcileRemoteConfigSchemas_AllSeedsEveryVersion(t *testing.T) {
	t.Parallel()

	deps, _, _ := newTestDeps()
	dir := writeSchemas(t, deps.fs, allThreeSchemas())

	require.NoError(t, reconcileRemoteConfigSchemas(t.Context(), dir, config.RemoteConfigSchemaLoadAll, deps))

	assert.True(t, schemaExists(t, deps, "otelcol-0.129.0"))
	assert.True(t, schemaExists(t, deps, "otelcol-0.130.0"))
	assert.True(t, schemaExists(t, deps, "otelcol-contrib-0.130.0"))
}

func TestReconcileRemoteConfigSchemas_NoneSeedsNothing(t *testing.T) {
	t.Parallel()

	deps, _, _ := newTestDeps()
	dir := writeSchemas(t, deps.fs, allThreeSchemas())

	require.NoError(t, reconcileRemoteConfigSchemas(t.Context(), dir, config.RemoteConfigSchemaLoadNone, deps))

	assert.False(t, schemaExists(t, deps, "otelcol-0.130.0"))
	assert.False(t, schemaExists(t, deps, "otelcol-contrib-0.130.0"))
}

func TestReconcileRemoteConfigSchemas_MissingDirSkips(t *testing.T) {
	t.Parallel()

	deps, _, _ := newTestDeps()

	require.NoError(t, reconcileRemoteConfigSchemas(
		t.Context(), "/does-not-exist", config.RemoteConfigSchemaLoadLatest, deps))
	assert.False(t, schemaExists(t, deps, "otelcol-0.130.0"))
}

func TestReconcileRemoteConfigSchemas_EmptyDirSkips(t *testing.T) {
	t.Parallel()

	deps, _, _ := newTestDeps()

	require.NoError(t, reconcileRemoteConfigSchemas(
		t.Context(), "", config.RemoteConfigSchemaLoadLatest, deps))
}

func TestReconcileRemoteConfigSchemas_RejectsUnexpectedKind(t *testing.T) {
	t.Parallel()

	deps, _, _ := newTestDeps()
	dir := writeSchemas(t, deps.fs, map[string]string{"bad.yaml": namespaceManifest})

	err := reconcileRemoteConfigSchemas(t.Context(), dir, config.RemoteConfigSchemaLoadAll, deps)
	require.ErrorIs(t, err, errUnsupportedKind)
}

func TestReconcileRemoteConfigSchemas_Idempotent(t *testing.T) {
	t.Parallel()

	deps, _, _ := newTestDeps()
	dir := writeSchemas(t, deps.fs, allThreeSchemas())

	require.NoError(t, reconcileRemoteConfigSchemas(t.Context(), dir, config.RemoteConfigSchemaLoadAll, deps))
	require.NoError(t, reconcileRemoteConfigSchemas(t.Context(), dir, config.RemoteConfigSchemaLoadAll, deps))

	// A second identical reconcile must not rewrite: ResourceVersion stays at its
	// post-create value of 1.
	schema, err := deps.schemaUsecase.GetRemoteConfigSchema(t.Context(), "default", "otelcol-0.130.0", nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), schema.Metadata.ResourceVersion)
}

func TestReconcileRemoteConfigSchemas_UpdatesOnChange(t *testing.T) {
	t.Parallel()

	deps, _, _ := newTestDeps()

	dir := writeSchemas(t, deps.fs, map[string]string{"otelcol-0.130.0.yaml": schemaOtelcol130})
	require.NoError(t, reconcileRemoteConfigSchemas(t.Context(), dir, config.RemoteConfigSchemaLoadAll, deps))

	// The library file changes its component catalog; the next reconcile updates in place.
	const changed = `kind: RemoteConfigSchema
apiVersion: v1
metadata:
  name: otelcol-0.130.0
  namespace: default
spec:
  binary: otelcol
  version: 0.130.0
  components:
    receivers:
      otlp:
        type: otlp
    exporters:
      debug:
        type: debug
`

	require.NoError(t, afero.WriteFile(
		deps.fs, filepath.Join(schemaDir, "otelcol-0.130.0.yaml"), []byte(changed), 0o600))
	require.NoError(t, reconcileRemoteConfigSchemas(t.Context(), dir, config.RemoteConfigSchemaLoadAll, deps))

	schema, err := deps.schemaUsecase.GetRemoteConfigSchema(t.Context(), "default", "otelcol-0.130.0", nil)
	require.NoError(t, err)
	assert.Equal(t, int64(2), schema.Metadata.ResourceVersion, "changed manifest triggers exactly one update")
	assert.Contains(t, schema.Spec.Components["exporters"], "debug")
}

func TestApplyManifests_RoutesRemoteConfigSchemaKind(t *testing.T) {
	t.Parallel()

	deps, _, _ := newTestDeps()
	dir := writeManifests(t, deps.fs, map[string]string{"schema.yaml": schemaOtelcol130})

	docs, err := loadManifestDocs(deps.fs, dir)
	require.NoError(t, err)

	require.NoError(t, applyManifests(t.Context(), docs, deps))
	assert.True(t, schemaExists(t, deps, "otelcol-0.130.0"))
}

func TestResolveRemoteConfigSchemaSettings(t *testing.T) {
	t.Parallel()

	// Default: dir derived from Dir, policy latest.
	//exhaustruct:ignore
	dir, load := resolveRemoteConfigSchemaSettings(&config.BootstrapSettings{Dir: "/etc/init"})
	assert.Equal(t, filepath.Join("/etc/init", "remoteconfigschema"), dir)
	assert.Equal(t, config.RemoteConfigSchemaLoadLatest, load)

	// Explicit overrides win.
	//exhaustruct:ignore
	dir, load = resolveRemoteConfigSchemaSettings(&config.BootstrapSettings{
		Dir:                    "/etc/init",
		RemoteConfigSchemaDir:  "/custom/schemas",
		RemoteConfigSchemaLoad: config.RemoteConfigSchemaLoadAll,
	})
	assert.Equal(t, "/custom/schemas", dir)
	assert.Equal(t, config.RemoteConfigSchemaLoadAll, load)

	// No dirs at all: schema dir stays empty (seeding skipped).
	//exhaustruct:ignore
	dir, _ = resolveRemoteConfigSchemaSettings(&config.BootstrapSettings{})
	assert.Empty(t, dir)
}
