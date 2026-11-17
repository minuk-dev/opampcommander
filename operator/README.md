# OpAMP Commander Operator

Kubebuilder bootstrap for the Kubernetes control plane described in [#896](https://github.com/minuk-dev/opampcommander/issues/896).
This is a separate Go module; the existing Commander API server is not changed.

## Current scope

- Namespaced `AgentGroup` and `RemoteConfig` under `opampcommander.io/v1alpha1`.
- Generated structural CRDs, deepcopy methods, scoped RBAC rules, Kustomize deployment and samples.
- A manager with health/readiness endpoints, optional leader election and two controllers.
- Controllers only set `Ready=Unknown`, reason `NotImplemented`, at the observed generation. They do not resolve references, select Agents, validate collector configuration, access MongoDB/Redis, or deliver configuration.

These alpha types are an initial subset, not a frozen migration contract. `RemoteConfig.spec.value` is for non-secret documents only; Secret references and admission validation follow in #903/#904. Do not use this scaffold to manage production Agents.

## Reproduce scaffolding

Generated using Kubebuilder **v4.16.0**, `go/v4`, Go module `github.com/minuk-dev/opampcommander/operator`:

```sh
kubebuilder init --domain io --repo github.com/minuk-dev/opampcommander/operator --project-name opampcommander-operator --license none
kubebuilder create api --group opampcommander --version v1alpha1 --kind AgentGroup --resource --controller
kubebuilder create api --group opampcommander --version v1alpha1 --kind RemoteConfig --resource --controller
```

The repository's MIT license applies. The scaffold pins controller-runtime v0.25.0 / Kubernetes libraries v0.37.0; envtest selects Kubernetes 1.37. This is a tested bootstrap baseline, not a declared older-cluster compatibility guarantee.

## Develop and verify

From `operator/`:

```sh
make manifests generate
make lint
make test       # integration-tagged envtest, with race detector; downloads local test binaries
make build
make build-installer IMG=ghcr.io/minuk-dev/opampcommander-operator:dev
```

The root Go module's tests do not traverse this nested module. `.github/workflows/operator-ci.yaml` runs its checks independently and detects stale generated manifests.

## Try in a disposable cluster

Check your kubeconfig context first. These commands install cluster-scoped CRDs and run the manager against that cluster:

```sh
make install
make run
# In another terminal:
kubectl apply -k config/samples
kubectl get agentgroups.opampcommander.io,remoteconfigs.opampcommander.io
kubectl get agentgroup collectors -o yaml
```

Both resources should report `Ready: Unknown`; this is not evidence of Agent application. Group selectors retain Commander's exact-match identifying/non-identifying attribute maps. An explicit empty selector is reserved to match all Agents in the namespace once delivery is implemented. References are same-namespace; filenames identify OpAMP config-map entries.

For an in-cluster manager, build/push your own image and run `make deploy IMG=...`. The committed `dist/install.yaml` uses a placeholder `:dev` image; no container image is published by this bootstrap. Regenerate it with your image before installing it. `make undeploy` removes that manager deployment. `make uninstall` deletes the CRDs **and their custom resources**; use only in a disposable environment.

## Next steps

#897 owns the complete ownership/tenancy ADR, #898 the remaining API contract, #899 dependency watches/runtime integration, and #901 delivery/acknowledgment. This bootstrap only contributes to those issues; none is complete yet. Ordinary configuration CRUD belongs to native kubectl. The later `kubectl opamp` plugin (#906) handles Agent operations through Commander.
