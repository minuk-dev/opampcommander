package infrastructure

import (
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	inmemory "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/inmemory"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

// manifestDir is the in-memory directory test manifests are written to.
const manifestDir = "/manifests"

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
		fs:                        afero.NewMemMapFs(),
		namespaceUsecase:          nsService,
		rolePersistencePort:       roleRepo,
		permissionPersistencePort: permRepo,
		clk:                       clock.NewRealClock(),
		logger:                    slog.Default(),
	}, roleRepo, permRepo
}

// writeManifests writes the given files into manifestDir on fs and returns the dir.
func writeManifests(t *testing.T, fs afero.Fs, files map[string]string) string {
	t.Helper()

	for name, content := range files {
		require.NoError(t, afero.WriteFile(fs, filepath.Join(manifestDir, name), []byte(content), 0o600))
	}

	return manifestDir
}

func TestApplyManifests_SeedsNamespaceRoleAndPermissions(t *testing.T) {
	t.Parallel()

	deps, roleRepo, permRepo := newTestDeps()
	dir := writeManifests(t, deps.fs, map[string]string{
		"00-namespace.yaml": namespaceManifest,
		"10-role.yaml":      roleManifest,
	})

	docs, err := loadManifestDocs(deps.fs, dir)
	require.NoError(t, err)
	require.Len(t, docs, 2)

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

	deps, roleRepo, _ := newTestDeps()
	dir := writeManifests(t, deps.fs, map[string]string{"10-role.yaml": roleManifest})

	docs, err := loadManifestDocs(deps.fs, dir)
	require.NoError(t, err)

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

func TestApplyManifests_AssignsDeterministicUIDsToBuiltins(t *testing.T) {
	t.Parallel()

	// Apply the same manifest into two independent fresh stores, mimicking two
	// apiserver replicas booting against an empty DB at the same time.
	seed := func() (any, any) {
		deps, roleRepo, permRepo := newTestDeps()
		dir := writeManifests(t, deps.fs, map[string]string{"10-role.yaml": roleManifest})

		docs, err := loadManifestDocs(deps.fs, dir)
		require.NoError(t, err)
		require.NoError(t, applyManifests(t.Context(), docs, deps))

		role, err := roleRepo.GetRoleByName(t.Context(), "default")
		require.NoError(t, err)
		perm, err := permRepo.GetPermissionByName(t.Context(), "agent:GET")
		require.NoError(t, err)

		return role.Metadata.UID, perm.Metadata.UID
	}

	role1, perm1 := seed()
	role2, perm2 := seed()

	// Built-in resources get a name-derived UID, so both replicas write the same _id;
	// the unique _id index then collapses the concurrent inserts into one document
	// instead of creating duplicate roles/permissions that share a name.
	assert.Equal(t, builtinRoleUID("default"), role1)
	assert.Equal(t, role1, role2, "default role UID must be deterministic across startups")
	assert.Equal(t, builtinPermissionUID("agent:GET"), perm1)
	assert.Equal(t, perm1, perm2, "permission UID must be deterministic across startups")
}

func TestApplyManifests_UnknownKindFails(t *testing.T) {
	t.Parallel()

	deps, _, _ := newTestDeps()
	dir := writeManifests(t, deps.fs, map[string]string{
		"bad.yaml": "kind: Widget\napiVersion: v1\n",
	})

	docs, err := loadManifestDocs(deps.fs, dir)
	require.NoError(t, err)

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

	deps, _, _ := newTestDeps()
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

	deps, roleRepo, _ := newTestDeps()
	dir := writeManifests(t, deps.fs, map[string]string{
		"00-namespace.yaml": namespaceManifest,
		"10-role.yaml":      roleManifest,
	})

	docs, err := loadManifestDocs(deps.fs, dir)
	require.NoError(t, err)

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

func TestReconcileManifests_SeedsFromDir(t *testing.T) {
	t.Parallel()

	deps, roleRepo, _ := newTestDeps()
	dir := writeManifests(t, deps.fs, map[string]string{"10-role.yaml": customRoleManifest})

	require.NoError(t, reconcileManifests(t.Context(), dir, deps))

	role, err := roleRepo.GetRoleByName(t.Context(), "custom")
	require.NoError(t, err, "role from the configured manifest dir must be seeded")
	assert.True(t, role.Spec.IsBuiltIn)
}

func TestReconcileManifests_EmptyDirSkips(t *testing.T) {
	t.Parallel()

	deps, roleRepo, _ := newTestDeps()

	require.NoError(t, reconcileManifests(t.Context(), "", deps))

	_, err := roleRepo.GetRoleByName(t.Context(), "default")
	require.Error(t, err, "an empty dir must seed nothing")
}

func TestReconcileManifests_MissingDirSkipsWithoutError(t *testing.T) {
	t.Parallel()

	deps, _, _ := newTestDeps()

	// A non-empty but absent dir must not fail startup; it is a warn-and-skip.
	err := reconcileManifests(t.Context(), "/does-not-exist", deps)
	require.NoError(t, err, "a missing manifest dir must not fail startup")
}

func TestReconcileManifests_DirIsAFileSkipsWithoutError(t *testing.T) {
	t.Parallel()

	deps, roleRepo, _ := newTestDeps()

	// bootstrap.dir pointing at a regular file (a misconfiguration) must warn-and-skip,
	// not hard-fail startup.
	require.NoError(t, afero.WriteFile(deps.fs, "/not-a-dir", []byte("oops"), 0o600))

	require.NoError(t, reconcileManifests(t.Context(), "/not-a-dir", deps))

	_, err := roleRepo.GetRoleByName(t.Context(), "default")
	require.Error(t, err, "nothing must be seeded when dir is a file")
}
