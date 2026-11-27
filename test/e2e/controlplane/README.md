# Control-plane E2E

Run this suite **only in CI**, through the control-plane E2E job in
`.github/workflows/test.yaml`. The job runs this command from the repository root:

```sh
make test-e2e-controlplane
```

The CI runner provides Docker, Kind v0.33.0, kubectl and the repository's Go toolchain. The
runner creates a uniquely named Kubernetes 1.37.0 Kind cluster with its own
kubeconfig. It never reuses an existing cluster or changes the user's context.
It builds the current apiserver and Operator, runs the existing Operator
health/metrics suite, and then deploys apiserver, Operator, MongoDB and Redis
together. Test credentials and ephemeral storage are only for this cluster.

Logs, events and Kind diagnostics go to `build/e2e-controlplane` (override with
`E2E_ARTIFACTS_DIR`). The runner deletes only its own cluster on success or
failure. No Docker-wide cleanup is performed.

## Regression coverage during migration

The `test` workflow runs all three E2E jobs on every PR. The Kind job supplements
the existing runtime suites; it does not replace them with a smaller smoke test.

| Contract | Existing regression suite | Combined Kind coverage |
| --- | --- | --- |
| Agent registration, description, health, heartbeats | `apiserver/e2e_test.go`, `conformance_e2e_test.go` | Reference Agents over HTTP and WebSocket report health/sequence through the deployed apiserver. |
| Remote/effective config and restart | `conformance_e2e_test.go`, `agentgroup_remoteconfig_e2e_test.go` | Existing REST selection and config/report/restart round trips through the deployed apiserver. |
| Agent persistence failures and Redis data loss | OpAMP persistence unit tests, Redis runtime E2E | Mongo update failures return `Unavailable` over HTTP/WebSocket without publishing the failed report; reports resume after recovery. Redis `FLUSHDB` rebuilds liveness while preserving committed reports and configuration. |
| CR configuration and dependency changes | Operator envtest API/dependency suites | All five samples, server-side dry-run with no persisted resources, a denied AgentGroup create with no persistence, no-op apply, and missing/restored dependency conditions. |
| Operator deployment and authenticated metrics | `operator/test/e2e` | The existing manager suite runs on this same disposable Kind cluster. |
| Restart and ownership | Operator dependency envtest | Changes made while the Operator is stopped converge after startup; two replicas use a Lease and recover after the leader Pod is deleted. Gateway ping remains available. |
| Installation and namespace ownership | Operator identity unit tests and namespace-watch envtest | Unassigned/foreign namespaces remain untouched. Assignment enqueues existing CRs. Recreating the installation ConfigMap makes both replicas unready and prevents further status writes; gateway ping remains available. |
| Runtime namespace identity | Runtime controller and HTTP routing unit tests | Checks assigned and foreign namespaces, same-name namespace recreation, and rejection of replacement installation UIDs against the original MongoDB database. |
| Selectors, namespace boundaries, CRUD/deletion, inventory, certificates, mTLS, Redis outage, Kafka | Remaining `test/e2e/apiserver/*_test.go` cases | Retained in `test-e2e`; these fault-injection cases are not yet ported to Kind. |
| Released Supervisor connection-settings policy | `test-supervisor-policy` | Retained as its own workflow job. |

There is **no CR-to-Agent delivery assertion yet**. The Operator currently checks
reference existence and reports `Ready=Unknown`; apiserver still uses its legacy
configuration path. The REST round trip above guards existing behavior and must
not be presented as proof of CR delivery or Mongo configuration removal.

Follow-up #900/#901 should add the CR apply → selected Agent desired revision →
OpAMP offer → acknowledgment assertion here before switching storage authority.
Cover an AgentGroup that matches one Agent and a group action spanning multiple
Agents. Verify that users with Agent read access can inspect their namespace and
that all user-requested changes go through an authorized AgentGroup write.
Keep the existing selector/priority, namespace isolation, deletion, reconnect and
failure cases until equivalent tests pass on the new path. #902–#907 add auth,
Secret rotation, validation, finalizers and migration/rollback cases to this
shared deployed environment; do not weaken current expectations to enable them.

Mongo failpoints are enabled only in this disposable test deployment. The write
failure scenario rejects update commands while leaving reads available; it does
not simulate a network partition or verify cross-gateway session fencing.
