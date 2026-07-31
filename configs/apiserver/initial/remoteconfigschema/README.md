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
