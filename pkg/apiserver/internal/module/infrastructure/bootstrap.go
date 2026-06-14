package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/afero"
	"go.uber.org/fx"
	"gopkg.in/yaml.v3"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/port"
	usermodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user/model"
	userport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user/port"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

// ErrInvalidPermissionName is returned when a role references a permission name that
// is not in the expected "resource:action" form.
var ErrInvalidPermissionName = errors.New("invalid permission name (want resource:action)")

// errUnsupportedKind is returned when a manifest declares a kind the bootstrap
// reconciler does not know how to apply.
var errUnsupportedKind = errors.New("unsupported manifest kind")

// errUnsupportedAPIVersion is returned when a manifest declares an apiVersion the
// bootstrap reconciler does not understand.
var errUnsupportedAPIVersion = errors.New("unsupported manifest apiVersion")

// errEmptyNamespaceName / errEmptyRoleName are returned when a manifest omits the
// identifying name field.
var (
	errEmptyNamespaceName = errors.New("namespace manifest has empty metadata.name")
	errEmptyRoleName      = errors.New("role manifest has empty spec.displayName")
)

// bootstrapUIDNamespace is a fixed UUID namespace used to derive deterministic UIDs
// for built-in resources from their names. Roles and permissions are keyed by UID in
// persistence but looked up by name, and the get-by-name / put sequence is not atomic;
// deriving the UID from the name means concurrent apiserver startups against a fresh
// database write the same _id and the unique index collapses them into a single record
// instead of inserting duplicate roles/permissions that share a name but differ by UID.
const bootstrapUIDNamespace = "7c9e6679-7425-40de-944b-e07fc1f90ae7"

// builtinRoleUID / builtinPermissionUID derive a stable UID for a built-in resource
// from its name (UUIDv5), so creating it is idempotent across processes and restarts.
func builtinRoleUID(name string) uuid.UUID {
	return uuid.NewSHA1(uuid.MustParse(bootstrapUIDNamespace), []byte("role/"+name))
}

func builtinPermissionUID(name string) uuid.UUID {
	return uuid.NewSHA1(uuid.MustParse(bootstrapUIDNamespace), []byte("permission/"+name))
}

// bootstrapDeps bundles the filesystem and persistence ports the manifest appliers
// reconcile into. fs is abstracted via afero so tests run against an in-memory
// filesystem instead of the real OS, and production uses afero.NewOsFs().
type bootstrapDeps struct {
	fs                        afero.Fs
	namespaceUsecase          agentport.NamespaceUsecase
	rolePersistencePort       userport.RolePersistencePort
	permissionPersistencePort userport.PermissionPersistencePort
	clk                       clock.PassiveClock
	logger                    *slog.Logger
}

// manifestDoc is a single decoded YAML document together with its type meta and
// source (filename) for error context. The payload is held as JSON because the
// api/v1 types are tagged with `json` only (no `yaml` tags), so decoding goes
// YAML -> generic -> JSON -> target.
type manifestDoc struct {
	kind       string
	apiVersion string
	json       []byte
	source     string
}

type typeMeta struct {
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`
}

// loadManifestDocs reads every *.yaml / *.yml file under dir on fsys (sorted by
// filename so numeric prefixes control apply order), splitting multi-document files
// on "---".
func loadManifestDocs(fsys afero.Fs, dir string) ([]manifestDoc, error) {
	entries, err := afero.ReadDir(fsys, dir)
	if err != nil {
		return nil, fmt.Errorf("read manifest dir %q: %w", dir, err)
	}

	names := make([]string, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext == ".yaml" || ext == ".yml" {
			names = append(names, entry.Name())
		}
	}

	sort.Strings(names)

	var docs []manifestDoc

	for _, name := range names {
		data, err := afero.ReadFile(fsys, filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("read manifest %q: %w", name, err)
		}

		fileDocs, err := decodeManifestBytes(data, name)
		if err != nil {
			return nil, err
		}

		docs = append(docs, fileDocs...)
	}

	return docs, nil
}

// decodeManifestBytes decodes all YAML documents in a single file's bytes into manifestDoc.
func decodeManifestBytes(data []byte, source string) ([]manifestDoc, error) {
	var docs []manifestDoc

	decoder := yaml.NewDecoder(bytes.NewReader(data))

	for {
		var generic any

		err := decoder.Decode(&generic)
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("parse yaml %q: %w", source, err)
		}

		if generic == nil {
			continue // empty document
		}

		jsonBytes, err := json.Marshal(generic)
		if err != nil {
			return nil, fmt.Errorf("re-encode %q as json: %w", source, err)
		}

		var meta typeMeta

		err = json.Unmarshal(jsonBytes, &meta)
		if err != nil {
			return nil, fmt.Errorf("read type meta from %q: %w", source, err)
		}

		docs = append(docs, manifestDoc{
			kind:       meta.Kind,
			apiVersion: meta.APIVersion,
			json:       jsonBytes,
			source:     source,
		})
	}

	return docs, nil
}

// applyManifests reconciles every manifest document into persistence. Reconciliation
// is declarative (full overwrite): the manifest is the source of truth, so on every
// startup built-in resources are reset to match it.
func applyManifests(ctx context.Context, docs []manifestDoc, deps bootstrapDeps) error {
	for _, doc := range docs {
		if doc.apiVersion != v1.APIVersion {
			return fmt.Errorf("%w: %q in %q (want %q)",
				errUnsupportedAPIVersion, doc.apiVersion, doc.source, v1.APIVersion)
		}

		var err error

		switch doc.kind {
		case v1.NamespaceKind:
			err = applyNamespace(ctx, doc, deps)
		case v1.RoleKind:
			err = applyRole(ctx, doc, deps)
		default:
			return fmt.Errorf("%w: kind %q in %q", errUnsupportedKind, doc.kind, doc.source)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

// applyNamespace upserts a namespace, preserving the existing CreatedAt/conditions
// when the namespace already exists and overwriting labels/annotations from the manifest.
func applyNamespace(ctx context.Context, doc manifestDoc, deps bootstrapDeps) error {
	var apiNamespace v1.Namespace

	err := json.Unmarshal(doc.json, &apiNamespace)
	if err != nil {
		return fmt.Errorf("decode Namespace from %q: %w", doc.source, err)
	}

	name := apiNamespace.Metadata.Name
	if name == "" {
		return fmt.Errorf("%w: %q", errEmptyNamespaceName, doc.source)
	}

	existing, err := deps.namespaceUsecase.GetNamespace(ctx, name, nil)
	if err != nil && !errors.Is(err, port.ErrResourceNotExist) {
		return fmt.Errorf("check namespace %q: %w", name, err)
	}

	if errors.Is(err, port.ErrResourceNotExist) {
		deps.logger.Info("bootstrap: creating namespace", slog.String("name", name))

		namespace := agentmodel.NewNamespace(name)
		namespace.MarkAsCreated(deps.clk.Now(), "system")
		namespace.Metadata.Labels = apiNamespace.Metadata.Labels
		namespace.Metadata.Annotations = apiNamespace.Metadata.Annotations

		_, err = deps.namespaceUsecase.SaveNamespace(ctx, namespace)
		if err != nil {
			return fmt.Errorf("save namespace %q: %w", name, err)
		}

		return nil
	}

	// Already exists: only re-save when the manifest actually changes labels/annotations,
	// to avoid a redundant write on every startup. maps.Equal treats nil and empty alike.
	if maps.Equal(existing.Metadata.Labels, apiNamespace.Metadata.Labels) &&
		maps.Equal(existing.Metadata.Annotations, apiNamespace.Metadata.Annotations) {
		return nil
	}

	existing.Metadata.Labels = apiNamespace.Metadata.Labels
	existing.Metadata.Annotations = apiNamespace.Metadata.Annotations

	_, err = deps.namespaceUsecase.SaveNamespace(ctx, existing)
	if err != nil {
		return fmt.Errorf("save namespace %q: %w", name, err)
	}

	return nil
}

// applyRole upserts a role, setting its permission list to exactly the manifest's
// (full overwrite). Permission objects referenced by name are auto-created from the
// "resource:action" encoding so SyncPolicies can resolve them.
func applyRole(ctx context.Context, doc manifestDoc, deps bootstrapDeps) error {
	var apiRole v1.Role

	err := json.Unmarshal(doc.json, &apiRole)
	if err != nil {
		return fmt.Errorf("decode Role from %q: %w", doc.source, err)
	}

	name := apiRole.Spec.DisplayName
	if name == "" {
		return fmt.Errorf("%w: %q", errEmptyRoleName, doc.source)
	}

	for _, permName := range apiRole.Spec.Permissions {
		err := ensurePermission(ctx, permName, deps)
		if err != nil {
			return err
		}
	}

	desiredPermissions := append([]string(nil), apiRole.Spec.Permissions...)

	existing, err := deps.rolePersistencePort.GetRoleByName(ctx, name)
	if err != nil && !errors.Is(err, port.ErrResourceNotExist) {
		return fmt.Errorf("check role %q: %w", name, err)
	}

	var role *usermodel.Role

	if errors.Is(err, port.ErrResourceNotExist) {
		deps.logger.Info("bootstrap: creating role", slog.String("name", name))

		role = usermodel.NewRole(name, apiRole.Spec.IsBuiltIn)
		// Deterministic UID so two concurrent fresh-DB startups converge on one record.
		role.Metadata.UID = builtinRoleUID(name)
	} else {
		// Already exists: skip the write (and UpdatedAt bump) when nothing changed,
		// to avoid churning the role on every startup.
		if existing.Spec.Description == apiRole.Spec.Description &&
			existing.Spec.IsBuiltIn == apiRole.Spec.IsBuiltIn &&
			slices.Equal(existing.Spec.Permissions, desiredPermissions) {
			return nil
		}

		role = existing
		role.Metadata.UpdatedAt = deps.clk.Now()
	}

	role.Spec.Description = apiRole.Spec.Description
	role.Spec.IsBuiltIn = apiRole.Spec.IsBuiltIn
	role.Spec.Permissions = desiredPermissions

	_, err = deps.rolePersistencePort.PutRole(ctx, role)
	if err != nil {
		return fmt.Errorf("save role %q: %w", name, err)
	}

	return nil
}

// ensurePermission creates the built-in permission object for name ("resource:action")
// if it does not already exist. Permissions are immutable once created.
func ensurePermission(ctx context.Context, name string, deps bootstrapDeps) error {
	resource, action, ok := strings.Cut(name, ":")
	if !ok || resource == "" || action == "" {
		return fmt.Errorf("%w: %q", ErrInvalidPermissionName, name)
	}

	_, err := deps.permissionPersistencePort.GetPermissionByName(ctx, name)
	if err == nil {
		return nil
	}

	if !errors.Is(err, port.ErrResourceNotExist) {
		return fmt.Errorf("check permission %q: %w", name, err)
	}

	deps.logger.Info("bootstrap: creating built-in permission", slog.String("name", name))

	permission := usermodel.NewPermission(resource, action, true)
	permission.Spec.Description = "Built-in: " + action + " access to " + resource
	// Deterministic UID so two concurrent fresh-DB startups converge on one record.
	permission.Metadata.UID = builtinPermissionUID(name)

	_, err = deps.permissionPersistencePort.PutPermission(ctx, permission)
	if err != nil {
		return fmt.Errorf("create permission %q: %w", name, err)
	}

	return nil
}

// registerBootstrapHook reconciles the initial manifests under BootstrapSettings.Dir
// into persistence on startup. When Dir is empty, reconciliation is skipped.
func registerBootstrapHook(
	lifecycle fx.Lifecycle,
	namespaceUsecase agentport.NamespaceUsecase,
	rolePersistencePort userport.RolePersistencePort,
	permissionPersistencePort userport.PermissionPersistencePort,
	settings *config.ServerSettings,
	logger *slog.Logger,
) {
	deps := bootstrapDeps{
		fs:                        afero.NewOsFs(),
		namespaceUsecase:          namespaceUsecase,
		rolePersistencePort:       rolePersistencePort,
		permissionPersistencePort: permissionPersistencePort,
		clk:                       clock.NewRealClock(),
		logger:                    logger,
	}

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return reconcileManifests(ctx, settings.BootstrapSettings.Dir, deps)
		},
		OnStop: nil,
	})
}

// reconcileManifests applies the initial manifests from dir. dir is the directory of
// default manifests an operator can inspect, edit, or point elsewhere via
// bootstrap.dir. An empty dir disables seeding; a dir that does not exist or is not a
// directory logs a warning and skips (never a hard boot failure). A present-but-
// malformed manifest still returns an error.
func reconcileManifests(ctx context.Context, dir string, deps bootstrapDeps) error {
	if dir == "" {
		deps.logger.Info("bootstrap: no manifest dir configured, skipping")

		return nil
	}

	isDir, err := afero.DirExists(deps.fs, dir)
	if err != nil {
		return fmt.Errorf("stat manifest dir %q: %w", dir, err)
	}

	if !isDir {
		deps.logger.Warn("bootstrap: manifest dir does not exist or is not a directory, skipping",
			slog.String("dir", dir),
		)

		return nil
	}

	docs, err := loadManifestDocs(deps.fs, dir)
	if err != nil {
		return err
	}

	deps.logger.Info("bootstrap: applying initial manifests",
		slog.String("dir", dir),
		slog.Int("documents", len(docs)),
	)

	return applyManifests(ctx, docs, deps)
}
