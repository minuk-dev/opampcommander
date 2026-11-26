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
	"reflect"
	"sort"
	"strings"

	"github.com/spf13/afero"
	"go.uber.org/fx"
	"gopkg.in/yaml.v3"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/helper"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

// errUnsupportedKind is returned when a manifest declares a kind the bootstrap
// reconciler does not know how to apply.
var errUnsupportedKind = errors.New("unsupported manifest kind")

// errUnsupportedAPIVersion is returned when a manifest declares an apiVersion the
// bootstrap reconciler does not understand.
var errUnsupportedAPIVersion = errors.New("unsupported manifest apiVersion")

// errEmptyNamespaceName / errEmptyRoleName / errEmptyUser* are returned when a manifest
// omits a required identifying field.
var (
	errEmptyNamespaceName     = errors.New("namespace manifest has empty metadata.name")
	errEmptyEndpointName      = errors.New("endpoint manifest has empty metadata.name")
	errEmptyEndpointNamespace = errors.New("endpoint manifest has empty metadata.namespace")

	errEmptyRemoteConfigSchemaName      = errors.New("remote config schema manifest has empty metadata.name")
	errEmptyRemoteConfigSchemaNamespace = errors.New("remote config schema manifest has empty metadata.namespace")
)

// bootstrapDeps bundles the filesystem and persistence ports the manifest appliers
// reconcile into. fs is abstracted via afero so tests run against an in-memory
// filesystem instead of the real OS, and production uses afero.NewOsFs().
type bootstrapDeps struct {
	fs               afero.Fs
	namespaceUsecase agentport.NamespaceUsecase
	endpointUsecase  agentport.EndpointUsecase
	schemaUsecase    agentport.RemoteConfigSchemaUsecase
	clk              clock.PassiveClock
	logger           *slog.Logger
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
		case v1.EndpointKind:
			err = applyEndpoint(ctx, doc, deps)
		case v1.RemoteConfigSchemaKind:
			err = applyRemoteConfigSchemaDoc(ctx, doc, deps)
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

	for attempt := 0; ; attempt++ {
		err = reconcileBootstrapNamespace(ctx, deps, &apiNamespace)
		if err == nil || !errors.Is(err, model.ErrConflict) || attempt >= bootstrapConflictRetries {
			return err
		}
	}
}

// reconcileBootstrapNamespace re-reads the namespace after a concurrent bootstrap write wins.
func reconcileBootstrapNamespace(ctx context.Context, deps bootstrapDeps, desired *v1.Namespace) error {
	name := desired.Metadata.Name

	existing, err := deps.namespaceUsecase.GetNamespace(ctx, name, nil)
	if err != nil && !errors.Is(err, model.ErrResourceNotExist) {
		return fmt.Errorf("check namespace %q: %w", name, err)
	}

	if errors.Is(err, model.ErrResourceNotExist) {
		deps.logger.Info("bootstrap: creating namespace", slog.String("name", name))

		namespace := agentmodel.NewNamespace(name)
		namespace.MarkAsCreated(deps.clk.Now(), "system")
		namespace.Metadata.Labels = desired.Metadata.Labels
		namespace.Metadata.Annotations = desired.Metadata.Annotations

		_, err = deps.namespaceUsecase.SaveNamespace(ctx, namespace)
		if err != nil {
			return fmt.Errorf("save namespace %q: %w", name, err)
		}

		return nil
	}

	// Already exists: only re-save when the manifest actually changes labels/annotations,
	// to avoid a redundant write on every startup. maps.Equal treats nil and empty alike.
	if maps.Equal(existing.Metadata.Labels, desired.Metadata.Labels) &&
		maps.Equal(existing.Metadata.Annotations, desired.Metadata.Annotations) {
		return nil
	}

	existing.Metadata.Labels = desired.Metadata.Labels
	existing.Metadata.Annotations = desired.Metadata.Annotations

	_, err = deps.namespaceUsecase.SaveNamespace(ctx, existing)
	if err != nil {
		return fmt.Errorf("save namespace %q: %w", name, err)
	}

	return nil
}

// applyEndpoint upserts an endpoint from a manifest. Like the other bootstrap
// resources it is declarative: a missing endpoint is created with a Created
// condition, and an existing one has its spec (URL/protocol/signals/tenants/
// metricsQuery) and attributes overwritten from the manifest while its CreatedAt
// and conditions are preserved. Redundant writes on unchanged manifests are
// skipped so startup does not churn the store.
func applyEndpoint(ctx context.Context, doc manifestDoc, deps bootstrapDeps) error {
	var apiEndpoint v1.Endpoint

	err := json.Unmarshal(doc.json, &apiEndpoint)
	if err != nil {
		return fmt.Errorf("decode Endpoint from %q: %w", doc.source, err)
	}

	name := apiEndpoint.Metadata.Name
	if name == "" {
		return fmt.Errorf("%w: %q", errEmptyEndpointName, doc.source)
	}

	namespace := apiEndpoint.Metadata.Namespace
	if namespace == "" {
		return fmt.Errorf("%w: %q", errEmptyEndpointNamespace, doc.source)
	}

	desired := helper.NewMapper(deps.clk, 0).MapAPIToEndpoint(&apiEndpoint)

	// The endpoint is versioned (optimistic concurrency), so a concurrent apiserver
	// startup applying the same manifest can lose the compare-and-swap and get
	// model.ErrConflict. Retry by re-reading: the winner has already written the
	// desired spec, so the next pass converges (equal -> skip) instead of failing
	// startup. Bootstrap must stay idempotent across concurrent HA processes.
	for attempt := 0; ; attempt++ {
		err = reconcileBootstrapEndpoint(ctx, deps, namespace, name, desired)
		if err == nil || !errors.Is(err, model.ErrConflict) || attempt >= bootstrapConflictRetries {
			return err
		}
	}
}

// bootstrapConflictRetries bounds how many times a bootstrap resource write is
// re-read and retried after losing an optimistic-concurrency race to another
// apiserver applying the same manifest.
const bootstrapConflictRetries = 3

// reconcileBootstrapEndpoint creates the endpoint when absent, or updates it in
// place when the manifest changes its spec/attributes, leaving it untouched when
// already in the desired state. It may return a wrapped model.ErrConflict, which
// applyEndpoint retries.
func reconcileBootstrapEndpoint(
	ctx context.Context, deps bootstrapDeps, namespace, name string, desired *agentmodel.Endpoint,
) error {
	existing, err := deps.endpointUsecase.GetEndpoint(ctx, namespace, name, nil)
	if err != nil && !errors.Is(err, model.ErrResourceNotExist) {
		return fmt.Errorf("check endpoint %q/%q: %w", namespace, name, err)
	}

	if errors.Is(err, model.ErrResourceNotExist) {
		deps.logger.Info("bootstrap: creating endpoint",
			slog.String("namespace", namespace), slog.String("name", name))

		endpoint := agentmodel.NewEndpoint(namespace, name, desired.Metadata.Attributes, deps.clk.Now(), "system")
		endpoint.Spec = desired.Spec

		_, err = deps.endpointUsecase.SaveEndpoint(ctx, endpoint)
		if err != nil {
			return fmt.Errorf("save endpoint %q/%q: %w", namespace, name, err)
		}

		return nil
	}

	// Already exists: only re-save when the manifest actually changes the spec or
	// attributes, to avoid a redundant write on every startup.
	if reflect.DeepEqual(existing.Spec, desired.Spec) &&
		maps.Equal(existing.Metadata.Attributes, desired.Metadata.Attributes) {
		return nil
	}

	existing.Spec = desired.Spec
	existing.Metadata.Attributes = desired.Metadata.Attributes

	_, err = deps.endpointUsecase.SaveEndpoint(ctx, existing)
	if err != nil {
		return fmt.Errorf("save endpoint %q/%q: %w", namespace, name, err)
	}

	return nil
}

// applyRemoteConfigSchemaDoc decodes a RemoteConfigSchema manifest document and applies it.
func applyRemoteConfigSchemaDoc(ctx context.Context, doc manifestDoc, deps bootstrapDeps) error {
	var apiSchema v1.RemoteConfigSchema

	err := json.Unmarshal(doc.json, &apiSchema)
	if err != nil {
		return fmt.Errorf("decode RemoteConfigSchema from %q: %w", doc.source, err)
	}

	return applyRemoteConfigSchema(ctx, &apiSchema, doc.source, deps)
}

// applyRemoteConfigSchema upserts a RemoteConfigSchema. Like the other bootstrap
// resources it is declarative: a missing schema is created with a Created condition,
// and an existing one has its spec (binary/version/components) and attributes
// overwritten from the manifest while its CreatedAt and conditions are preserved.
// Redundant writes on unchanged manifests are skipped. Optimistic-concurrency
// conflicts from a concurrent HA startup applying the same schema are retried by
// re-reading, matching the endpoint applier.
func applyRemoteConfigSchema(
	ctx context.Context, apiSchema *v1.RemoteConfigSchema, source string, deps bootstrapDeps,
) error {
	name := apiSchema.Metadata.Name
	if name == "" {
		return fmt.Errorf("%w: %q", errEmptyRemoteConfigSchemaName, source)
	}

	namespace := apiSchema.Metadata.Namespace
	if namespace == "" {
		return fmt.Errorf("%w: %q", errEmptyRemoteConfigSchemaNamespace, source)
	}

	desired := helper.NewMapper(deps.clk, 0).MapAPIToRemoteConfigSchema(apiSchema)

	for attempt := 0; ; attempt++ {
		err := reconcileBootstrapRemoteConfigSchema(ctx, deps, namespace, name, desired)
		if err == nil || !errors.Is(err, model.ErrConflict) || attempt >= bootstrapConflictRetries {
			return err
		}
	}
}

// reconcileBootstrapRemoteConfigSchema creates the schema when absent, or updates it
// in place when the manifest changes its spec/attributes, leaving it untouched when
// already in the desired state. It may return a wrapped model.ErrConflict, which
// applyRemoteConfigSchema retries.
func reconcileBootstrapRemoteConfigSchema(
	ctx context.Context, deps bootstrapDeps, namespace, name string, desired *agentmodel.RemoteConfigSchema,
) error {
	existing, err := deps.schemaUsecase.GetRemoteConfigSchema(ctx, namespace, name, nil)
	if err != nil && !errors.Is(err, model.ErrResourceNotExist) {
		return fmt.Errorf("check remote config schema %q/%q: %w", namespace, name, err)
	}

	if errors.Is(err, model.ErrResourceNotExist) {
		deps.logger.Info("bootstrap: creating remote config schema",
			slog.String("namespace", namespace), slog.String("name", name))

		schema := agentmodel.NewRemoteConfigSchema(
			namespace, name, desired.Metadata.Attributes, deps.clk.Now(), "system")
		schema.Spec = desired.Spec

		_, err = deps.schemaUsecase.SaveRemoteConfigSchema(ctx, schema)
		if err != nil {
			return fmt.Errorf("save remote config schema %q/%q: %w", namespace, name, err)
		}

		return nil
	}

	// Already exists: only re-save when the manifest actually changes the spec or
	// attributes, to avoid a redundant write on every startup.
	if reflect.DeepEqual(existing.Spec, desired.Spec) &&
		maps.Equal(existing.Metadata.Attributes, desired.Metadata.Attributes) {
		return nil
	}

	existing.ApplyUpdate(desired)

	_, err = deps.schemaUsecase.SaveRemoteConfigSchema(ctx, existing)
	if err != nil {
		return fmt.Errorf("save remote config schema %q/%q: %w", namespace, name, err)
	}

	return nil
}

// reconcileRemoteConfigSchemas seeds the pre-built RemoteConfigSchema library from
// dir according to policy: "latest" seeds only the newest version per distribution,
// "all" seeds every version, "none" disables seeding. dir is loaded separately from
// the main manifest reconciler, which ignores subdirectories.
func reconcileRemoteConfigSchemas(ctx context.Context, dir, policy string, deps bootstrapDeps) error {
	if policy == config.RemoteConfigSchemaLoadNone {
		deps.logger.Info("bootstrap: remote config schema seeding disabled")

		return nil
	}

	if dir == "" {
		return nil
	}

	isDir, err := afero.DirExists(deps.fs, dir)
	if err != nil {
		return fmt.Errorf("stat remote config schema dir %q: %w", dir, err)
	}

	if !isDir {
		deps.logger.Warn("bootstrap: remote config schema dir does not exist, skipping",
			slog.String("dir", dir))

		return nil
	}

	docs, err := loadManifestDocs(deps.fs, dir)
	if err != nil {
		return err
	}

	schemas, err := decodeRemoteConfigSchemaDocs(docs)
	if err != nil {
		return err
	}

	if policy == config.RemoteConfigSchemaLoadLatest {
		schemas = latestRemoteConfigSchemaPerBinary(schemas)
	}

	deps.logger.Info("bootstrap: seeding remote config schemas",
		slog.String("dir", dir),
		slog.String("policy", policy),
		slog.Int("count", len(schemas)),
	)

	for _, apiSchema := range schemas {
		err = applyRemoteConfigSchema(ctx, apiSchema, apiSchema.Metadata.Name, deps)
		if err != nil {
			return err
		}
	}

	return nil
}

// decodeRemoteConfigSchemaDocs decodes every document in the schema library dir into a
// RemoteConfigSchema, rejecting any manifest whose apiVersion/kind is unexpected.
func decodeRemoteConfigSchemaDocs(docs []manifestDoc) ([]*v1.RemoteConfigSchema, error) {
	schemas := make([]*v1.RemoteConfigSchema, 0, len(docs))

	for _, doc := range docs {
		if doc.apiVersion != v1.APIVersion {
			return nil, fmt.Errorf("%w: %q in %q (want %q)",
				errUnsupportedAPIVersion, doc.apiVersion, doc.source, v1.APIVersion)
		}

		if doc.kind != v1.RemoteConfigSchemaKind {
			return nil, fmt.Errorf("%w: kind %q in %q (remote config schema dir expects %q)",
				errUnsupportedKind, doc.kind, doc.source, v1.RemoteConfigSchemaKind)
		}

		var apiSchema v1.RemoteConfigSchema

		err := json.Unmarshal(doc.json, &apiSchema)
		if err != nil {
			return nil, fmt.Errorf("decode RemoteConfigSchema from %q: %w", doc.source, err)
		}

		schemas = append(schemas, &apiSchema)
	}

	return schemas, nil
}

// latestRemoteConfigSchemaPerBinary keeps only the highest-versioned schema for each
// Spec.Binary, comparing Spec.Version as semver. The result is ordered by binary name
// so seeding is deterministic.
func latestRemoteConfigSchemaPerBinary(schemas []*v1.RemoteConfigSchema) []*v1.RemoteConfigSchema {
	latest := make(map[string]*v1.RemoteConfigSchema)

	for _, schema := range schemas {
		current, ok := latest[schema.Spec.Binary]
		if !ok || agentmodel.CompareCollectorVersion(schema.Spec.Version, current.Spec.Version) > 0 {
			latest[schema.Spec.Binary] = schema
		}
	}

	binaries := make([]string, 0, len(latest))
	for binary := range latest {
		binaries = append(binaries, binary)
	}

	sort.Strings(binaries)

	out := make([]*v1.RemoteConfigSchema, 0, len(binaries))
	for _, binary := range binaries {
		out = append(out, latest[binary])
	}

	return out
}

// registerBootstrapHook reconciles the initial manifests under BootstrapSettings.Dir
// into persistence on startup, then seeds the pre-built RemoteConfigSchema library.
// When Dir is empty, reconciliation is skipped.
func registerBootstrapHook(
	lifecycle fx.Lifecycle,
	namespaceUsecase agentport.NamespaceUsecase,
	endpointUsecase agentport.EndpointUsecase,
	schemaUsecase agentport.RemoteConfigSchemaUsecase,
	settings *config.ServerSettings,
	logger *slog.Logger,
) {
	deps := bootstrapDeps{
		fs:               afero.NewOsFs(),
		namespaceUsecase: namespaceUsecase,
		endpointUsecase:  endpointUsecase,
		schemaUsecase:    schemaUsecase,
		clk:              clock.NewRealClock(),
		logger:           logger,
	}

	schemaDir, schemaLoad := resolveRemoteConfigSchemaSettings(&settings.BootstrapSettings)

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			err := reconcileManifests(ctx, settings.BootstrapSettings.Dir, deps)
			if err != nil {
				return err
			}

			return reconcileRemoteConfigSchemas(ctx, schemaDir, schemaLoad, deps)
		},
		OnStop: nil,
	})
}

// resolveRemoteConfigSchemaSettings derives the effective schema library directory and
// load policy. The directory defaults to the "remoteconfigschema" subdirectory of the
// manifest dir when unset; the policy defaults to "latest".
func resolveRemoteConfigSchemaSettings(settings *config.BootstrapSettings) (string, string) {
	dir := settings.RemoteConfigSchemaDir
	if dir == "" && settings.Dir != "" {
		dir = filepath.Join(settings.Dir, "remoteconfigschema")
	}

	load := settings.RemoteConfigSchemaLoad
	if load == "" {
		load = config.RemoteConfigSchemaLoadLatest
	}

	return dir, load
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
