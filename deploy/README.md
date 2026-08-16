# deploy

| Path | What it is |
|---|---|
| [`charts/opampcommander`](charts/opampcommander) | Helm chart for the apiserver and web dashboard — the supported way to install on Kubernetes |
| [`examples/`](examples) | Example RBAC resources (roles, role bindings) applied through the API |
| [`sample/`](sample) | Hand-written manifests kept as an illustration of the moving parts; not maintained against releases — use the chart instead |

```sh
helm install opampcommander ./charts/opampcommander \
  --namespace opampcommander --create-namespace
```

See the [chart README](charts/opampcommander/README.md) for configuration, and
the [deployment guide](https://minuk-dev.github.io/opampcommander/docs/deployment/) for the
walkthrough.
