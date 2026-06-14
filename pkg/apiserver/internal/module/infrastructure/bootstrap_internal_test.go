package infrastructure

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	inmemory "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/inmemory"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

const namespaceManifest = `kind: Namespace
apiVersion: v1
metadata:
  name: default
`

const roleManifest = `kind: Role
apiVersion: v1
spec:
  displayName: default
  description: built-in
  isBuiltIn: true
  permissions:
    - agent:GET
    - agent:LIST
    - rolebinding:GET
`

const customRoleManifest = `kind: Role
apiVersion: v1
spec:
  displayName: custom
  isBuiltIn: true
  permissions:
    - agent:GET
`

// fixedClock is a clock.PassiveClock that always reports a fixed time, used to make
// a spurious write detectable via the timestamp it would stamp.
type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time                  { return c.now }
func (c fixedClock) Since(t time.Time) time.Duration { return c.now.Sub(t) }

func newTestDeps() (bootstrapDeps, *inmemory.RoleRepository, *inmemory.PermissionRepository) {
	roleRepo := inmemory.NewRoleRepository()
	permRepo := inmemory.NewPermissionRepository()
	nsService := agentservice.NewNamespaceService(inmemory.NewNamespaceRepository())

	return bootstrapDeps{
		namespaceUsecase:          nsService,
		rolePersistencePort:       roleRepo,
		permissionPersistencePort: permRepo,
		clk:                       clock.NewRealClock(),
		logger:                    slog.Default(),
	}, roleRepo, permRepo
}

func writeManifests(t *testing.T, files map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
	}

	return dir
}

func TestApplyManifests_SeedsNamespaceRoleAndPermissions(t *testing.T) {
	t.Parallel()

	dir := writeManifests(t, map[string]string{
		"00-namespace.yaml": namespaceManifest,
		"10-role.yaml":      roleManifest,
	})

	docs, err := loadManifestDocs(os.DirFS(dir))
	require.NoError(t, err)
	require.Len(t, docs, 2)

	deps, roleRepo, permRepo := newTestDeps()
	require.NoError(t, applyManifests(t.Context(), docs, deps))

	ns, err := deps.namespaceUsecase.GetNamespace(t.Context(), "default", nil)
	require.NoError(t, err)
	assert.Equal(t, "default", ns.Metadata.Name)

	role, err := roleRepo.GetRoleByName(t.Context(), "default")
	require.NoError(t, err)
	assert.True(t, role.Spec.IsBuiltIn)
	assert.Equal(t, []string{"agent:GET", "agent:LIST", "rolebinding:GET"}, role.Spec.Permissions)

	// Permission objects are auto-derived from "resource:action" names.
	perm, err := permRepo.GetPermissionByName(t.Context(), "agent:GET")
	require.NoError(t, err)
	assert.Equal(t, "agent", perm.Spec.Resource)
	assert.Equal(t, "GET", perm.Spec.Action)
	assert.True(t, perm.Spec.IsBuiltIn)
}

func TestApplyManifests_FullOverwriteResetsRolePermissions(t *testing.T) {
	t.Parallel()

	dir := writeManifests(t, map[string]string{"10-role.yaml": roleManifest})

	docs, err := loadManifestDocs(os.DirFS(dir))
	require.NoError(t, err)

	deps, roleRepo, _ := newTestDeps()
	require.NoError(t, applyManifests(t.Context(), docs, deps))

	// Simulate an admin adding an extra permission via the API.
	role, err := roleRepo.GetRoleByName(t.Context(), "default")
	require.NoError(t, err)
	role.AddPermission("certificate:GET")
	_, err = roleRepo.PutRole(t.Context(), role)
	require.NoError(t, err)

	// Re-applying the manifest must reset the permission list to the manifest's.
	require.NoError(t, applyManifests(t.Context(), docs, deps))

	role, err = roleRepo.GetRoleByName(t.Context(), "default")
	require.NoError(t, err)
	assert.Equal(t, []string{"agent:GET", "agent:LIST", "rolebinding:GET"}, role.Spec.Permissions)
	assert.False(t, role.HasPermission("certificate:GET"), "admin-added permission must be reset")
}

func TestApplyManifests_UnknownKindFails(t *testing.T) {
	t.Parallel()

	dir := writeManifests(t, map[string]string{
		"bad.yaml": "kind: Widget\napiVersion: v1\n",
	})

	docs, err := loadManifestDocs(os.DirFS(dir))
	require.NoError(t, err)

	deps, _, _ := newTestDeps()
	err = applyManifests(t.Context(), docs, deps)
	require.ErrorIs(t, err, errUnsupportedKind)
}

func TestEnsurePermission_RejectsMalformedName(t *testing.T) {
	t.Parallel()

	deps, _, _ := newTestDeps()
	err := ensurePermission(t.Context(), "notavalidname", deps)
	require.ErrorIs(t, err, ErrInvalidPermissionName)
}

func TestApplyManifests_RejectsUnsupportedAPIVersion(t *testing.T) {
	t.Parallel()

	dir := writeManifests(t, map[string]string{
		"ns.yaml": "kind: Namespace\napiVersion: v2\nmetadata:\n  name: default\n",
	})

	docs, err := loadManifestDocs(os.DirFS(dir))
	require.NoError(t, err)

	deps, _, _ := newTestDeps()
	err = applyManifests(t.Context(), docs, deps)
	require.ErrorIs(t, err, errUnsupportedAPIVersion)
}

func TestApplyManifests_IdempotentNoWriteWhenUnchanged(t *testing.T) {
	t.Parallel()

	dir := writeManifests(t, map[string]string{
		"00-namespace.yaml": namespaceManifest,
		"10-role.yaml":      roleManifest,
	})

	docs, err := loadManifestDocs(os.DirFS(dir))
	require.NoError(t, err)

	deps, roleRepo, _ := newTestDeps()
	require.NoError(t, applyManifests(t.Context(), docs, deps))

	role, err := roleRepo.GetRoleByName(t.Context(), "default")
	require.NoError(t, err)

	originalUpdatedAt := role.Metadata.UpdatedAt

	// Re-apply with a clock pinned far in the future: a spurious write would stamp
	// UpdatedAt with that time, so equality proves the unchanged manifest was a no-op.
	deps.clk = fixedClock{now: time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)}
	require.NoError(t, applyManifests(t.Context(), docs, deps))

	role, err = roleRepo.GetRoleByName(t.Context(), "default")
	require.NoError(t, err)
	assert.Equal(t, originalUpdatedAt, role.Metadata.UpdatedAt,
		"unchanged role must not be re-written (UpdatedAt must not move)")
}

func TestReconcileManifests_EmptyDirUsesEmbeddedManifests(t *testing.T) {
	t.Parallel()

	deps, roleRepo, _ := newTestDeps()

	require.NoError(t, reconcileManifests(t.Context(), "", deps))

	// The embedded manifests must have seeded the built-in default namespace and role.
	ns, err := deps.namespaceUsecase.GetNamespace(t.Context(), "default", nil)
	require.NoError(t, err)
	assert.Equal(t, "default", ns.Metadata.Name)

	role, err := roleRepo.GetRoleByName(t.Context(), "default")
	require.NoError(t, err)
	assert.True(t, role.Spec.IsBuiltIn)
	assert.True(t, role.HasPermission("agent:GET"), "embedded default role must carry its permissions")
}

func TestReconcileManifests_MissingOverrideDirFallsBackToEmbedded(t *testing.T) {
	t.Parallel()

	deps, roleRepo, _ := newTestDeps()

	// A non-empty but absent override directory must not fail startup; it falls back
	// to the embedded manifests.
	err := reconcileManifests(t.Context(), filepath.Join(t.TempDir(), "does-not-exist"), deps)
	require.NoError(t, err)

	_, err = roleRepo.GetRoleByName(t.Context(), "default")
	require.NoError(t, err, "embedded default role must still be seeded on fallback")
}

func TestReconcileManifests_OverrideDirReplacesEmbedded(t *testing.T) {
	t.Parallel()

	// An existing override dir wins over the embedded manifests: only the override's
	// resources are seeded.
	dir := writeManifests(t, map[string]string{"10-role.yaml": customRoleManifest})

	deps, roleRepo, _ := newTestDeps()
	require.NoError(t, reconcileManifests(t.Context(), dir, deps))

	_, err := roleRepo.GetRoleByName(t.Context(), "custom")
	require.NoError(t, err, "override role must be seeded")

	_, err = roleRepo.GetRoleByName(t.Context(), "default")
	require.Error(t, err, "embedded default role must NOT be seeded when an override dir is used")
}
