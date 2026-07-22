package infrastructure

import (
	"context"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	inmemory "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/inmemory"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	usermodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/security"
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

// testPepper is a fixed pepper so the test hasher is enabled (basic auth on).
const testPepper = "test-pepper"

func newTestDeps() (bootstrapDeps, *inmemory.RoleRepository, *inmemory.PermissionRepository) {
	roleRepo := inmemory.NewRoleRepository()
	permRepo := inmemory.NewPermissionRepository()
	userRepo := inmemory.NewUserRepository()
	// Bootstrap only exercises Get/Save on the namespace usecase, so the cascade
	// collaborators (sibling usecases + transaction port) are left nil here.
	nsService := agentservice.NewNamespaceService(
		inmemory.NewNamespaceRepository(),
		nil, nil, nil, nil,
		nil,
		agentmodel.DefaultNamespaceName,
	)

	//exhaustruct:ignore
	hasher := security.NewPasswordHasher(&security.Config{
		BasicAuthSettings: security.BasicAuthSettings{Pepper: testPepper},
	})

	return bootstrapDeps{
		fs:                        afero.NewMemMapFs(),
		namespaceUsecase:          nsService,
		endpointUsecase:           agentservice.NewEndpointService(inmemory.NewEndpointRepository()),
		rolePersistencePort:       roleRepo,
		permissionPersistencePort: permRepo,
		userPersistencePort:       userRepo,
		passwordHasher:            hasher,
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

const guestUserManifest = `kind: User
apiVersion: v1
spec:
  username: guest
  email: guest@guest
  isActive: true
  password: guest
`

func TestApplyManifests_SeedsBasicAuthUser(t *testing.T) {
	t.Parallel()

	deps, _, _ := newTestDeps()
	dir := writeManifests(t, deps.fs, map[string]string{"20-user.yaml": guestUserManifest})

	docs, err := loadManifestDocs(deps.fs, dir)
	require.NoError(t, err)

	require.NoError(t, applyManifests(t.Context(), docs, deps))

	user, err := deps.userPersistencePort.GetUserByUsername(t.Context(), "guest")
	require.NoError(t, err)
	assert.Equal(t, "guest@guest", user.Spec.Email)
	assert.True(t, user.Spec.IsActive)
	assert.Equal(t, builtinUserUID("guest"), user.Metadata.UID, "user UID must be deterministic")
	assert.True(t, user.HasBasicAuth())
	require.NotNil(t, user.GetIdentity(usermodel.IdentityProviderBasic))

	loginType, ok := user.GetLabel(usermodel.LabelLoginType)
	assert.True(t, ok)
	assert.Equal(t, usermodel.IdentityProviderBasic, loginType)

	// The seeded hash must verify against the manifest password (and not store the plaintext).
	require.NoError(t, deps.passwordHasher.Verify("guest", user.PasswordHash()))
	assert.NotContains(t, user.PasswordHash(), "guest", "hash must not embed the plaintext")
}

func TestApplyManifests_UserIdempotentNoWriteWhenUnchanged(t *testing.T) {
	t.Parallel()

	deps, _, _ := newTestDeps()
	dir := writeManifests(t, deps.fs, map[string]string{"20-user.yaml": guestUserManifest})

	docs, err := loadManifestDocs(deps.fs, dir)
	require.NoError(t, err)
	require.NoError(t, applyManifests(t.Context(), docs, deps))

	user, err := deps.userPersistencePort.GetUserByUsername(t.Context(), "guest")
	require.NoError(t, err)

	originalUpdatedAt := user.Metadata.UpdatedAt

	// Re-apply with a clock pinned in the future: a spurious re-hash/write would move UpdatedAt.
	deps.clk = fixedClock{now: time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)}
	require.NoError(t, applyManifests(t.Context(), docs, deps))

	user, err = deps.userPersistencePort.GetUserByUsername(t.Context(), "guest")
	require.NoError(t, err)
	assert.Equal(t, originalUpdatedAt, user.Metadata.UpdatedAt,
		"an unchanged user (password already matches) must not be re-written")
}

func TestApplyManifests_UserPasswordResetToManifest(t *testing.T) {
	t.Parallel()

	deps, _, _ := newTestDeps()
	dir := writeManifests(t, deps.fs, map[string]string{"20-user.yaml": guestUserManifest})

	docs, err := loadManifestDocs(deps.fs, dir)
	require.NoError(t, err)
	require.NoError(t, applyManifests(t.Context(), docs, deps))

	// Simulate the password being changed out-of-band to something other than the manifest's.
	user, err := deps.userPersistencePort.GetUserByUsername(t.Context(), "guest")
	require.NoError(t, err)

	otherHash, err := deps.passwordHasher.Hash("changed")
	require.NoError(t, err)
	user.SetPasswordHash(otherHash)
	_, err = deps.userPersistencePort.PutUser(t.Context(), user)
	require.NoError(t, err)

	// Re-applying the manifest must reset the password back to the declared one.
	require.NoError(t, applyManifests(t.Context(), docs, deps))

	user, err = deps.userPersistencePort.GetUserByUsername(t.Context(), "guest")
	require.NoError(t, err)
	require.NoError(t, deps.passwordHasher.Verify("guest", user.PasswordHash()),
		"manifest password must be the source of truth")
}

func TestApplyManifests_UserSkippedWhenBasicAuthDisabled(t *testing.T) {
	t.Parallel()

	deps, _, _ := newTestDeps()
	//exhaustruct:ignore
	deps.passwordHasher = security.NewPasswordHasher(&security.Config{}) // empty pepper => disabled

	dir := writeManifests(t, deps.fs, map[string]string{"20-user.yaml": guestUserManifest})

	docs, err := loadManifestDocs(deps.fs, dir)
	require.NoError(t, err)

	// Disabled basic auth must skip the user seed, not fail startup.
	require.NoError(t, applyManifests(t.Context(), docs, deps))

	_, err = deps.userPersistencePort.GetUserByUsername(t.Context(), "guest")
	require.Error(t, err, "user must not be seeded when basic auth is disabled")
}

func TestApplyManifests_UserRequiresPassword(t *testing.T) {
	t.Parallel()

	deps, _, _ := newTestDeps()
	dir := writeManifests(t, deps.fs, map[string]string{
		"20-user.yaml": "kind: User\napiVersion: v1\nspec:\n  username: nopass\n  email: nopass@x\n",
	})

	docs, err := loadManifestDocs(deps.fs, dir)
	require.NoError(t, err)

	err = applyManifests(t.Context(), docs, deps)
	require.Error(t, err, "a user manifest without a password must be rejected")
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

func TestApplyManifests_SeedsEndpointWithMetricsQuery(t *testing.T) {
	t.Parallel()

	deps, _, _ := newTestDeps()
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

	deps, _, _ := newTestDeps()
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

	deps, _, _ := newTestDeps()
	dir := writeManifests(t, deps.fs, map[string]string{
		"30-endpoint.yaml": "kind: Endpoint\napiVersion: v1\nmetadata:\n  name: vm\n",
	})

	docs, err := loadManifestDocs(deps.fs, dir)
	require.NoError(t, err)

	err = applyManifests(t.Context(), docs, deps)
	require.ErrorIs(t, err, errEmptyEndpointNamespace)
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
