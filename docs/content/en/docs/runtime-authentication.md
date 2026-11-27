---
title: Runtime installation and namespace identity
description: Configure the private runtime API and bind its data to a Kubernetes installation
---

Users manage Commander through the Kubernetes API with Kubernetes RBAC. The
Commander apiserver serves an internal REST API for trusted workloads and an
OpAMP endpoint for Agents. Protect internal routes at the network boundary.
If infrastructure exposes an internal route, enforce caller access there.

The Operator uses `GET /api/runtime/v1/namespaces/{name}` over HTTPS to verify
the Kubernetes identities of the installation and namespace:

```json
{"name":"collectors","namespaceUID":"<namespace UID>","installationUID":"<anchor ConfigMap UID>"}
```

## Apiserver setup

Run the apiserver inside Kubernetes with its own ServiceAccount. Configure the
three coordinates of the installed anchor:

```yaml
runtimeAPI:
  installationNamespace: opampcommander-operator-system
  installationName: opampcommander-operator-opamp-installation
  installationUID: "<existing ConfigMap metadata.uid>"
```

All three coordinates are required together. Leave them empty for deployments
without Kubernetes installation binding. The corresponding flags are
`--runtimeAPI.*`, with environment variables such as `RUNTIMEAPI_INSTALLATIONUID`.
Keep the configured UID across restarts and upgrades.

With MongoDB, the apiserver verifies the installation
before running schema migrations or serving traffic. On first startup with
`database.ddlAuto=true`, it records the configured UID in the unsharded
`runtime_metadata` collection using a majority write. With automatic DDL
disabled, provision this document in the apiserver database once, using the live
anchor UID and a majority write concern:

```javascript
db.runtime_metadata.insertOne({ _id: "installation", uid: "<anchor UID>" }, { writeConcern: { w: "majority" } })
```

Back up this document with the Agent data. If a restored cluster has a different
anchor UID, startup stops even when the new UID matches the apiserver configuration.
Recovery requires an explicit migration of installation identity. Recording the
installation does not enroll existing Agents or assign them to namespaces.

The apiserver ServiceAccount needs `get` on core Namespaces and `get` on the
anchor ConfigMap. Scope the ConfigMap permission to the anchor name in its
namespace. A request rechecks the anchor and the target Namespace's
`opampcommander.io/installation-uid` label. A missing Namespace returns 404.
An unassigned, foreign or terminating namespace returns 403. An unavailable
or replaced anchor returns 503. Recreating a namespace gives it a fresh UID.
Agent reads and work derived from AgentGroups validate the enrolled namespace
and installation UIDs independently.

## Read an Agent enrollment

On a Mongo-backed installation, the internal endpoint
`GET /api/runtime/v1/namespaces/{name}/agent-enrollments/{instanceUid}` reads a
stored enrollment scoped to the live namespace and installation UIDs. The
response includes the Agent instance UID and the ServiceAccount UID recorded
at enrollment. A lookup in a recreated namespace returns 404 for a record
bound to its previous UID. The runtime HTTP API provides read access for this
resource. User-directed changes follow the
[AgentGroup management contract](../crd-control-plane/#agent-management).

## Check your namespace from kubectl

Use `kubectl opamp runtime-scope` to check the namespace assignment against the
installed identity anchor. The command reads the Namespace and installation
ConfigMap through the Kubernetes API using the selected kubeconfig:

```sh
kubectl opamp runtime-scope --context staging --namespace collectors \
  --installation-namespace opampcommander-operator-system \
  --installation-name opampcommander-operator-opamp-installation -o json
```

With no `--namespace`, the command uses the context's namespace. Grant `get` on
the named core Namespace at cluster scope and `get` on the anchor ConfigMap in
its namespace. Restrict both grants with `resourceNames`. The result covers
Kubernetes ownership. Gateway readiness and MongoDB state are separate checks.
The output formats are `text` (default), `json`, and `yaml`. The same command is
available as `opampctl kube runtime-scope`.
