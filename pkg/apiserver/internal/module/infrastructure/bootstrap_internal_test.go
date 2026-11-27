package infrastructure

import (
	"context"
	"log/slog"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	inmemory "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/inmemory"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

// manifestDir is the in-memory directory test manifests are written to.
const manifestDir = "/manifests"

const namespaceManifest = `kind: Namespace
apiVersion: v1
metadata:
  name: default
`

func newTestDeps() bootstrapDeps {
	// Bootstrap only exercises Get/Save on the namespace usecase, so the cascade
	// collaborators (sibling usecases + transaction port) are left nil here.
	nsService := agentservice.NewNamespaceService(
		inmemory.NewNamespaceRepository(),
		nil, nil, nil, nil,
		nil,
		agentmodel.DefaultNamespaceName,
	)

	return bootstrapDeps{
		fs:               afero.NewMemMapFs(),
		namespaceUsecase: nsService,
		endpointUsecase:  agentservice.NewEndpointService(inmemory.NewEndpointRepository()),
		schemaUsecase:    agentservice.NewRemoteConfigSchemaService(inmemory.NewRemoteConfigSchemaRepository()),
		clk:              clock.NewRealClock(),
		logger:           slog.Default(),
	}
}

// writeManifests writes the given files into manifestDir on fs and returns the dir.
func writeManifests(t *testing.T, fs afero.Fs, files map[string]string) string {
	t.Helper()

	for name, content := range files {
		require.NoError(t, afero.WriteFile(fs, filepath.Join(manifestDir, name), []byte(content), 0o600))
	}

	return manifestDir
}

func TestApplyManifests_UnknownKindFails(t *testing.T) {
	t.Parallel()

	deps := newTestDeps()
	dir := writeManifests(t, deps.fs, map[string]string{
		"bad.yaml": "kind: Widget\napiVersion: v1\n",
	})

	docs, err := loadManifestDocs(deps.fs, dir)
	require.NoError(t, err)

	err = applyManifests(t.Context(), docs, deps)
	require.ErrorIs(t, err, errUnsupportedKind)
}

func TestApplyManifests_SeedsEndpointWithMetricsQuery(t *testing.T) {
	t.Parallel()

	deps := newTestDeps()
	endpointRepo := inmemory.NewEndpointRepository()
	deps.endpointUsecase = agentservice.NewEndpointService(endpointRepo)

	dir := writeManifests(t, deps.fs, map[string]string{
		"30-endpoint.yaml": `kind: Endpoint
apiVersion: v1
metadata:
  name: vm
  namespace: default
spec:
  url: http://vm/insert
  protocol: prometheusremotewrite
  signals:
    metrics: true
  metricsQuery:
    metrics: sum(rate(otelcol_exporter_sent_metric_points_total[{{.Window}}]))
`,
	})

	docs, err := loadManifestDocs(deps.fs, dir)
	require.NoError(t, err)
	require.NoError(t, applyManifests(t.Context(), docs, deps))

	got, err := endpointRepo.GetEndpoint(t.Context(), "default", "vm", nil)
	require.NoError(t, err)
	assert.Equal(t, "http://vm/insert", got.Spec.URL)
	assert.True(t, got.Spec.Signals.Metrics)
	require.NotNil(t, got.Spec.MetricsQuery)
	assert.Contains(t, got.Spec.MetricsQuery.Metrics, "otelcol_exporter_sent_metric_points_total")

	// Reconciliation is idempotent: a second apply must not error.
	require.NoError(t, applyManifests(t.Context(), docs, deps))
}

// conflictOnceEndpointUsecase wraps an EndpointUsecase and returns model.ErrConflict
// on the first SaveEndpoint call, simulating losing a bootstrap optimistic-
// concurrency race to a concurrent apiserver before succeeding on retry.
type conflictOnceEndpointUsecase struct {
	agentport.EndpointUsecase

	saveCalls int
}

func (w *conflictOnceEndpointUsecase) SaveEndpoint(
	ctx context.Context, endpoint *agentmodel.Endpoint,
) (*agentmodel.Endpoint, error) {
	w.saveCalls++
	if w.saveCalls == 1 {
		return nil, model.ErrConflict
	}

	return w.EndpointUsecase.SaveEndpoint(ctx, endpoint) //nolint:wrapcheck // test delegate
}

func TestApplyManifests_EndpointUpdateRetriesOnConflict(t *testing.T) {
	t.Parallel()

	deps := newTestDeps()
	endpointRepo := inmemory.NewEndpointRepository()
	delegate := agentservice.NewEndpointService(endpointRepo)

	// Seed an existing endpoint whose spec differs from the manifest, so bootstrap
	// takes the update path (the only one that can hit a version conflict).
	seeded := agentmodel.NewEndpoint("default", "vm", nil, deps.clk.Now(), "system")
	seeded.Spec.URL = "http://old/insert"
	_, err := delegate.SaveEndpoint(t.Context(), seeded)
	require.NoError(t, err)

	wrapper := &conflictOnceEndpointUsecase{EndpointUsecase: delegate}
	deps.endpointUsecase = wrapper

	dir := writeManifests(t, deps.fs, map[string]string{
		"30-endpoint.yaml": `kind: Endpoint
apiVersion: v1
metadata:
  name: vm
  namespace: default
spec:
  url: http://vm/insert
  protocol: prometheusremotewrite
  signals:
    metrics: true
`,
	})

	docs, err := loadManifestDocs(deps.fs, dir)
	require.NoError(t, err)

	// The first write loses the race; applyEndpoint must re-read and retry rather
	// than fail startup, so the manifest still converges.
	require.NoError(t, applyManifests(t.Context(), docs, deps))
	assert.GreaterOrEqual(t, wrapper.saveCalls, 2, "the conflicting write must be retried")

	got, err := endpointRepo.GetEndpoint(t.Context(), "default", "vm", nil)
	require.NoError(t, err)
	assert.Equal(t, "http://vm/insert", got.Spec.URL, "the desired spec must win after the retry")
}

func TestApplyManifests_EndpointRequiresNamespace(t *testing.T) {
	t.Parallel()

	deps := newTestDeps()
	dir := writeManifests(t, deps.fs, map[string]string{
		"30-endpoint.yaml": "kind: Endpoint\napiVersion: v1\nmetadata:\n  name: vm\n",
	})

	docs, err := loadManifestDocs(deps.fs, dir)
	require.NoError(t, err)

	err = applyManifests(t.Context(), docs, deps)
	require.ErrorIs(t, err, errEmptyEndpointNamespace)
}

func TestApplyManifests_RejectsUnsupportedAPIVersion(t *testing.T) {
	t.Parallel()

	deps := newTestDeps()
	dir := writeManifests(t, deps.fs, map[string]string{
		"ns.yaml": "kind: Namespace\napiVersion: v2\nmetadata:\n  name: default\n",
	})

	docs, err := loadManifestDocs(deps.fs, dir)
	require.NoError(t, err)

	err = applyManifests(t.Context(), docs, deps)
	require.ErrorIs(t, err, errUnsupportedAPIVersion)
}

func TestApplyManifests_IdempotentNoWriteWhenUnchanged(t *testing.T) {
	t.Parallel()

	deps := newTestDeps()
	dir := writeManifests(t, deps.fs, map[string]string{
		"00-namespace.yaml": namespaceManifest,
	})

	docs, err := loadManifestDocs(deps.fs, dir)
	require.NoError(t, err)

	require.NoError(t, applyManifests(t.Context(), docs, deps))

	ns, err := deps.namespaceUsecase.GetNamespace(t.Context(), "default", nil)
	require.NoError(t, err)

	originalVersion := ns.Metadata.ResourceVersion

	require.NoError(t, applyManifests(t.Context(), docs, deps))

	ns, err = deps.namespaceUsecase.GetNamespace(t.Context(), "default", nil)
	require.NoError(t, err)
	assert.Equal(t, originalVersion, ns.Metadata.ResourceVersion,
		"unchanged namespace must retain its version")
}

func TestReconcileManifests_SeedsFromDir(t *testing.T) {
	t.Parallel()

	deps := newTestDeps()
	dir := writeManifests(t, deps.fs, map[string]string{"00-namespace.yaml": namespaceManifest})

	require.NoError(t, reconcileManifests(t.Context(), dir, deps))

	ns, err := deps.namespaceUsecase.GetNamespace(t.Context(), "default", nil)
	require.NoError(t, err, "namespace from the configured manifest dir must be seeded")
	assert.Equal(t, "default", ns.Metadata.Name)
}

func TestReconcileManifests_EmptyDirSkips(t *testing.T) {
	t.Parallel()

	deps := newTestDeps()

	require.NoError(t, reconcileManifests(t.Context(), "", deps))

	_, err := deps.namespaceUsecase.GetNamespace(t.Context(), "default", nil)
	require.Error(t, err, "an empty dir must seed nothing")
}

func TestReconcileManifests_MissingDirSkipsWithoutError(t *testing.T) {
	t.Parallel()

	deps := newTestDeps()

	// A non-empty but absent dir must not fail startup; it is a warn-and-skip.
	err := reconcileManifests(t.Context(), "/does-not-exist", deps)
	require.NoError(t, err, "a missing manifest dir must not fail startup")
}

func TestReconcileManifests_DirIsAFileSkipsWithoutError(t *testing.T) {
	t.Parallel()

	deps := newTestDeps()

	// bootstrap.dir pointing at a regular file (a misconfiguration) must warn-and-skip,
	// not hard-fail startup.
	require.NoError(t, afero.WriteFile(deps.fs, "/not-a-dir", []byte("oops"), 0o600))

	require.NoError(t, reconcileManifests(t.Context(), "/not-a-dir", deps))

	_, err := deps.namespaceUsecase.GetNamespace(t.Context(), "default", nil)
	require.Error(t, err, "nothing must be seeded when dir is a file")
}

// barrierNamespaceUsecase makes both bootstrappers read the same version before either can write.
type barrierNamespaceUsecase struct {
	agentport.NamespaceUsecase

	reads   atomic.Int32
	ready   chan struct{}
	release chan struct{}
}

func (b *barrierNamespaceUsecase) GetNamespace(
	ctx context.Context, name string, opts *model.GetOptions,
) (*agentmodel.Namespace, error) {
	namespace, err := b.NamespaceUsecase.GetNamespace(ctx, name, opts)
	if b.reads.Add(1) <= 2 {
		b.ready <- struct{}{}

		<-b.release
	}

	return namespace, err //nolint:wrapcheck // test delegate
}

func TestApplyManifests_ConcurrentNamespaceBootstrap(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"create", "update"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			deps := newTestDeps()
			delegate := deps.namespaceUsecase
			expectedVersion := int64(1)

			if name == "update" {
				seed := agentmodel.NewNamespace("default")
				seed.Metadata.Labels = map[string]string{"old": "value"}
				_, err := delegate.SaveNamespace(t.Context(), seed)
				require.NoError(t, err)

				expectedVersion++
			}

			dir := writeManifests(t, deps.fs, map[string]string{"00-namespace.yaml": namespaceManifest})
			docs, err := loadManifestDocs(deps.fs, dir)
			require.NoError(t, err)

			barrier := &barrierNamespaceUsecase{
				NamespaceUsecase: delegate,
				ready:            make(chan struct{}, 2), release: make(chan struct{}),
			}
			deps.namespaceUsecase = barrier
			results := make(chan error, 2)

			for range 2 {
				go func() { results <- applyManifests(t.Context(), docs, deps) }()
			}

			<-barrier.ready
			<-barrier.ready
			close(barrier.release)

			for range 2 {
				require.NoError(t, <-results, "identical manifests must converge across concurrent servers")
			}

			stored, err := delegate.GetNamespace(t.Context(), "default", nil)
			require.NoError(t, err)
			assert.Empty(t, stored.Metadata.Labels)
			assert.Equal(t, expectedVersion, stored.Metadata.ResourceVersion, "only one mutation should commit")
		})
	}
}
