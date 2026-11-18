# Configuration API contract

The namespaced API is `opampcommander.io/v1alpha1`. The supported bootstrap
Kubernetes minor is **1.37**; CI runs envtest 1.37.0. Older minors and other
controller-runtime versions are not covered by this declaration. Production
install/upgrade qualification belongs to #908/#910.

The [ownership ADR](../docs/content/en/docs/crd-control-plane.md) defines the
target runtime behavior. These types establish configuration shape; they do not
switch existing Commander storage authority or implement delivery.

| Kind | Desired configuration |
| --- | --- |
| AgentGroup | Exact-match attribute selector, priority, filename-to-RemoteConfig references, package references and optional complete connection-settings Secret reference. |
| RemoteConfig | Exactly one public verbatim `value` or `valueFrom` Secret key, content type, schema and endpoint references. |
| AgentPackage | Package type/version, immutable HTTPS artifact URL/SHA-256/size, Secret-backed headers and optional signature reference. Package delivery remains unsupported. |
| RemoteConfigSchema | Collector distribution/version and an immutable external JSON catalog reference. |
| Endpoint | Manually managed destination, protocol, signals, tenant header Secret references and optional bounded query templates. Discovered endpoints remain runtime observations. |

There is no Agent CRD. Namespace, Secret, RBAC and ServiceAccount remain native
Kubernetes resources. Every reference is local to the CR namespace; names must
be valid Kubernetes resource names. A reference has no namespace override or
global fallback. Structural validation does not imply the target exists;
admission and reconcile-time resolution are implemented in #903/#904/#899.

## Selection and composition

All entries in both Agent attribute maps must match. `selector: {}` explicitly
selects all enrolled Agents in the namespace; omitting selector is invalid.
Matching groups compose configurations by OpAMP filename. Higher priority wins;
equal priority uses the lexicographically smallest Kubernetes group name. The
empty filename is valid and unique within a group, just like other keys.
The list-map schema prevents duplicate filenames and allows apply ownership per
entry. Connection settings select a complete bundle, not a field-by-field merge.

Package references declare intent only. Endpoint references record dependencies;
they do not enable implicit template substitution. Neither produces network side
effects in this API-only change. New resource kinds have no controller yet; an
absent Ready condition is not success. The existing two bootstrap controllers
still report Ready=Unknown/NotImplemented.

## Payloads and credentials

`RemoteConfig.value` is a string so Kubernetes cannot prune unknown collector
fields inside it. The API preserves its bytes, including vendor-specific keys.
The default content type is `application/yaml`; supported content types are
`application/yaml`, `text/yaml`, `application/json`, and `text/plain`. Semantic
collector validation is separate from the structural CRD schema.

Inline text is limited to **256 KiB UTF-8**, not 256k characters. CEL also checks
byte length, so multibyte text cannot bypass the limit. Secret-backed documents
must be resolved and checked against the same delivery budget by #903/#904;
the API cannot inspect Secret bytes. Empty text and ambiguous/missing sources
are rejected. Private text belongs in a Secret, not `value`.

Artifacts carry an immutable HTTPS URL, lowercase `sha256:` digest and positive
exact `sizeBytes`. Changing an artifact requires a new resource name. Catalogs
are capped at 16 MiB and package artifacts at 512 MiB; downloaders must enforce
the declared size, digest, timeout and network policy before using bytes.
No fetch is performed by these types. URLs have no embedded user credentials;
Endpoint URLs also exclude query strings and fragments. Header values and
signatures use Secret keys. Resolution/rotation and redaction are follow-up work.

Groups allow at most 64 configurations and 64 package references. RemoteConfigs
allow 16 schema references and 64 endpoint references. Endpoints allow 64 tenants,
32 headers per tenant and four distinct signals. Status Conditions are list-map
entries keyed by type with at most 16 entries. Do not store Agent membership
arrays, heartbeat timestamps or resolved credentials in status.

## Catalog storage measurement

The contract test loads the checked-in
`configs/apiserver/initial/remoteconfigschema/otelcol-contrib-0.157.0.yaml`,
converts it to JSON, and sends the component catalog through server-side apply
to a temporary test-only CRD with unknown fields preserved. A server-side dry-run
measures the API representation including managedFields. A real write verifies
the default etcd request limit rejects this full representation. A small catalog
then verifies that adding a second apply manager grows stored managedFields.

With envtest Kubernetes 1.37.0, the fixture measured 2,021,763 bytes of YAML,
950,765 bytes of legacy JSON, and 950,516 bytes for the components map. The
server-side dry-run returned 2,060,015 bytes, including 1,109,238 bytes of
managedFields for one apply manager; persisting it failed with
`etcdserver: request is too large`. Metadata lengths can vary between runs.

This reproduces why compact source JSON size alone is an unsafe storage budget.
The full catalog remains outside CRs. `RemoteConfigSchema.spec.catalog` identifies
the canonical JSON **components map**, not the legacy resource wrapper, by
URL/digest/size. Unknown vendor component fields remain intact in that artifact.
The schema API stores only its identity and resolution status.

To reproduce the measurement with local envtest binaries:

```sh
make setup-envtest
KUBEBUILDER_ASSETS="$(bin/setup-envtest use 1.37 --bin-dir bin -p path)" \
  go test -tags=integration ./internal/controller -v \
  -ginkgo.focus='representative catalog' -ginkgo.v
```

The `catalog-size` output records source YAML, legacy JSON, catalog JSON,
server-dry-run and managedFields byte lengths. This is a storage probe, not a
fleet-scale capacity benchmark.

## Kubernetes clients, writes and evolution

Generated CRDs and deepcopy code are checked in. Kubernetes clients use the
registered Go types with controller-runtime's client; a separate generated
clientset is unnecessary. The contract tests exercise this client for native
create/get/list/watch, update, status and server-side apply operations.

```go
scheme := runtime.NewScheme()
if err := v1alpha1.AddToScheme(scheme); err != nil {
    return err
}
kube, err := client.New(config, client.Options{Scheme: scheme})
// Use kube.Get/List/Create/Patch and kube.Status() with the registered API types.
```

ObjectMeta is native: resourceVersion stays an opaque string, and generation is
not a delivery acknowledgment. `/status` updates cannot change spec, ordinary
writes cannot inject status, and stale resourceVersion updates fail. Apply
managers conflict when changing a field owned by another manager; this API does
not force ownership. Ready columns expose reported Conditions without implying
that apply has reached an Agent.

Unknown fields outside the opaque configuration string follow Kubernetes
structural-schema pruning. Use strict field validation in automation to catch
misspelled or unsupported fields. Never interpret a pruned namespace property as
permission to cross namespace boundaries.

This is an alpha API. Optional compatible fields may be added; breaking shape
or interpretation changes require an explicit version/migration decision before
storing new data. Do not change list-map keys, reinterpret existing artifact
digests or copy Mongo resource versions into ObjectMeta. Existing bootstrap
inline configurations larger than 256 KiB must be split or moved through the
explicit migration path before this tighter schema is installed; no automatic
truncation or conversion is performed.

The examples in `config/samples` are structurally valid. Artifact URLs ending in
`example.invalid`, their digests and sizes are illustrative placeholders and
must be replaced before any resolver is enabled.
