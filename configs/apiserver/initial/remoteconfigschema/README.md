# Pre-built RemoteConfigSchema library

Each `*.yaml` here is a `kind: RemoteConfigSchema` manifest describing the component
catalog (receivers/processors/exporters/extensions/connectors) of one OTel Collector
distribution at one version — `otelcol`, `otelcol-contrib`, and `otelcol-k8s`.

These files are **generated**, not hand-edited. The component *type* names are read
from the compiled collector binary's `components` subcommand (they are not derivable
from release manifests), so regeneration downloads each release binary:

```sh
make gen-remoteconfigschema           # all stable releases, all distributions
VERSIONS="v0.130.0" hack/gen-remoteconfigschema.sh   # a specific version
```

On apiserver startup the library is seeded per `bootstrap.remoteConfigSchemaLoad`:

- `latest` (default) — seed only the newest version of each distribution
- `all` — seed every version in this directory
- `none` — seed nothing (the files stay on disk for opt-in use)

The main manifest reconciler ignores this subdirectory; it is loaded separately.

## Per-component config schemas (field-level validation)

A schema may also carry `spec.componentConfigs` — the config field tree
(field names + coarse types) of each component, enabling a config's component
settings to be validated (unknown keys, type mismatches), not just component
existence. Components without an entry keep existence-only validation.

Field names/types are only knowable from the components' Go config structs (there
is no machine-readable dump), so they are produced by reflecting the config types.
`hack/gen-component-configs.sh` compiles a distribution's component set and prints a
JSON catalog; `gen-remoteconfigschema.sh` merges it via `CONFIGS_DIR`:

```sh
# 1. reflect the config field schemas for a distribution/version (heavy; prefer CI)
mkdir -p /tmp/cfgcatalogs
hack/gen-component-configs.sh --release otelcol v0.157.0 > /tmp/cfgcatalogs/otelcol-0.157.0.json
# 2. regenerate the schema, merging the config catalog
CONFIGS_DIR=/tmp/cfgcatalogs FORCE=1 VERSIONS="v0.157.0" DISTS="otelcol" hack/gen-remoteconfigschema.sh
```

Compiling the component set is heavy (large module downloads), so populating config
schemas for every distribution/version is best done in CI. Currently the newest
`otelcol` and `otelcol-k8s` carry config schemas; the rest validate component
existence only until regenerated.
