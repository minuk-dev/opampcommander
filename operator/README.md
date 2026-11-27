# OpAMP Commander Operator

Kubebuilder bootstrap for the Kubernetes control plane described in [#896](https://github.com/minuk-dev/opampcommander/issues/896).
This is a separate Go module; the existing Commander API server is not changed.

The [ownership and migration ADR](../docs/content/en/docs/crd-control-plane.md)
defines the target resource, tenancy, consistency and compatibility contracts.

## Current scope

- Namespaced `AgentGroup`, `RemoteConfig`, `AgentPackage`, `RemoteConfigSchema`, and `Endpoint` under `opampcommander.io/v1alpha1`; see the [API contract](API.md).
- Generated structural CRDs, deepcopy methods, scoped RBAC rules, Kustomize deployment and samples.
- A manager with health/readiness endpoints, optional leader election and two controllers.
- An installation ConfigMap UID and explicit native Namespace assignment gate both controllers. Missing/replaced installation identity fails startup or readiness and stops reconciliation.
- AgentGroup watches its configuration dependencies through namespace-scoped reference indexes. It reports `ReferencesFound` after checking object existence; Secret contents are not cached or resolved.
- Controllers keep `Ready=Unknown`, reason `NotImplemented`, at the observed generation. They do not select Agents, validate collector configuration, access MongoDB/Redis, or deliver configuration.

These alpha types define configuration shape, not completed runtime behavior. `RemoteConfig.spec.value` is for non-secret documents only; Secret reference resolution and admission validation follow in #903/#904. New resource kinds have no controller yet. Do not use this scaffold to manage production Agents.

## Reproduce scaffolding

Generated using Kubebuilder **v4.16.0**, `go/v4`, Go module `github.com/minuk-dev/opampcommander/operator`:

```sh
kubebuilder init --domain io --repo github.com/minuk-dev/opampcommander/operator --project-name opampcommander-operator --license none
kubebuilder create api --group opampcommander --version v1alpha1 --kind AgentGroup --resource --controller
kubebuilder create api --group opampcommander --version v1alpha1 --kind RemoteConfig --resource --controller
kubebuilder create api --group opampcommander --version v1alpha1 --kind AgentPackage --resource --controller=false
kubebuilder create api --group opampcommander --version v1alpha1 --kind RemoteConfigSchema --resource --controller=false
kubebuilder create api --group opampcommander --version v1alpha1 --kind Endpoint --resource --controller=false
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

In CI only, the repository-root `make test-e2e-controlplane` runs the existing manager E2E
suite and the combined apiserver/Operator environment in a fresh Kind cluster;
see the [regression coverage](../test/e2e/controlplane/README.md).

## Configuration watches

The installer creates `opampcommander-operator-opamp-installation` in the
Operator namespace. Kustomize carries name-prefix changes into the Deployment's
ConfigMap argument. Its Kubernetes UID is the installation identity; editing
its data preserves identity, while deleting/recreating it does not. The manager
requires this object at startup and checks the original UID with an uncached
reader before reconciling and during readiness probes. It never adopts a new
UID in place. Preserve the ConfigMap during upgrades.

An administrator opts a namespace in by setting
`opampcommander.io/installation-uid` to that UID. Both controllers leave
unassigned, foreign and terminating namespaces untouched. Namespace label changes
enqueue existing CRs; periodic repair covers missed notifications. The manager
retains cluster-wide CR/metadata watches; this gate controls reconciliation, not
Kubernetes API read permissions. Restrict namespace-label writes with native RBAC.

Namespace reassignment requires a fenced migration (#907). Preserve
installation and namespace UIDs when connecting Commander.

AgentGroups index direct RemoteConfig, AgentPackage and connection Secret
references. RemoteConfigs index schema, endpoint and Secret references; packages
and endpoints index their Secret references. A dependency event follows these
indexes back to affected groups in the same namespace. Secret watches and reads
use metadata only. A missing or deleting dependency sets `ReferencesFound=False`;
read errors set Unknown and retry with controller-runtime backoff. This condition
does not establish that Secret keys exist or configuration contents are valid.

Status-only updates do not enqueue configuration work. Each reconcile rereads
the current graph, uses an optimistic status patch, and schedules a five-minute
repair for missed notifications. The standard deduplicating queue stores group
keys with two workers; memory scales with the number of referenced objects and
groups, not Agent membership. Informers handle watch reconnect/relist, and initial
cache contents enqueue existing groups after restart. Readiness waits for all
configured caches to synchronize and verifies the live installation anchor.
It does not establish freshness of every cached configuration object.

The installed deployment enables native Lease leader election. Gateway replicas
continue serving Agent sessions independently. This implements the Operator
input side of #899; the authenticated runtime boundary and retirement of legacy
Mongo configuration readers/bootstrap writers remain unfinished.

`make test-e2e` is CI-only and uses a disposable Kind cluster named `opampcommander-operator-test-e2e` (override with `KIND_CLUSTER`). It creates an isolated kubeconfig without changing your current context; kubectl kuberc is disabled. Direct E2E runs must also set `KIND_CLUSTER` and export that cluster's configuration successfully before tests can start. The E2E suite deletes its test resources and, on success, the Make target deletes the selected Kind cluster, including a reused cluster.

## Try in a disposable cluster

Check your kubeconfig context first. These commands install cluster-scoped CRDs and run the manager against that cluster:

```sh
make install
kubectl create configmap opamp-installation -n default
installation_uid=$(kubectl get configmap opamp-installation -n default -o jsonpath='{.metadata.uid}')
kubectl label namespace default "opampcommander.io/installation-uid=$installation_uid" --overwrite
POD_NAMESPACE=default make run
# In another terminal:
kubectl apply -k config/samples
kubectl get agentgroups.opampcommander.io,remoteconfigs.opampcommander.io
kubectl get agentgroup collectors -o yaml
```

Both resources should report `Ready: Unknown`; this is not evidence of Agent application. Group selectors retain Commander's exact-match identifying/non-identifying attribute maps. An explicit empty selector is reserved to match all Agents in the namespace once delivery is implemented. References are same-namespace; filenames identify OpAMP config-map entries.

For an in-cluster manager, build/push your own image and run `make deploy IMG=...`.
Assign the target namespace using the UID of the **installed** ConfigMap, rather
than the local-development ConfigMap above:

```sh
installation_uid=$(kubectl get configmap opampcommander-operator-opamp-installation -n opampcommander-operator-system -o jsonpath='{.metadata.uid}')
kubectl label namespace YOUR_NAMESPACE "opampcommander.io/installation-uid=$installation_uid" --overwrite
```

 The committed `dist/install.yaml` uses a placeholder `:dev` image; no container image is published by this bootstrap. Regenerate it with your image before installing it. `make undeploy` removes the manager and its installation ConfigMap; it is not an upgrade procedure. `make uninstall` deletes the CRDs **and their custom resources**; use only in a disposable environment.

## Next steps

The ownership/tenancy ADR (#897) and configuration API contract (#898) define the migration boundary. Dependency watches/runtime integration (#899), Agent storage (#900), delivery/acknowledgment (#901), and Secret resolution/admission (#903/#904) follow. Ordinary configuration CRUD belongs to native kubectl. The later `kubectl opamp` plugin (#906) handles Agent operations through Commander.

### Connect AgentGroup reconciliation to Commander

Configure the Operator with the HTTPS origin of the Commander Service via
`--runtime-url` or `COMMANDER_RUNTIME_URL`. With this option enabled, the
Operator calls `GET /api/runtime/v1/namespaces/{name}` when it reconciles an
AgentGroup in an assigned namespace. It compares the returned installation UID
and namespace UID with the live Kubernetes identities.

Adapt [the Deployment patch](config/runtime/connection-patch.yaml) to your
Commander Service and its certificate. Create the referenced CA ConfigMap in
the Operator namespace. The client uses system trust roots when no CA file
is specified. Allow the Operator to reach the private Commander Service over
the network. See [runtime identity setup](../docs/content/en/docs/runtime-authentication.md).

A successful lookup sets the AgentGroup condition `RuntimeScopeVerified=True`.
On failure it becomes `Unknown` and reconciliation retries. This condition
records the last check for the observed generation. It attests to runtime scope
only. Delivery and Agent acknowledgments are tracked separately; `Ready`
remains unknown until policy delivery is implemented. With the option disabled,
the controller omits the scope condition.
