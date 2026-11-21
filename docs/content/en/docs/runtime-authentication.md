---
title: Runtime authentication
description: Kubernetes identity and namespace discovery for the CRD runtime API
---

The opt-in native runtime endpoint resolves the installation and namespace UIDs
needed by runtime clients. `GET /api/runtime/v1/namespaces/{name}` returns:

```json
{"name":"collectors","namespaceUID":"<namespace UID>","installationUID":"<anchor ConfigMap UID>"}
```

This is identity discovery, not enrollment or Agent access. Legacy `/api/v1`
routes still use their existing JWT/Casbin policy. Native credentials are not
accepted by those routes, and legacy admin tokens cannot bypass native checks.
Agent records do not yet carry the required installation/namespace UID binding;
none are exposed through this endpoint. See the [control-plane ADR](../crd-control-plane/).

## Gateway setup

Run the gateway inside Kubernetes with its own ServiceAccount. It uses the
in-cluster CA and rotating ServiceAccount credential to call the Kubernetes API.
Enable the endpoint explicitly:

```yaml
runtimeAPI:
  enabled: true
  audience: opampcommander-runtime
  installationNamespace: opampcommander-operator-system
  installationName: opampcommander-operator-opamp-installation
  installationUID: "<existing ConfigMap metadata.uid>"
```

The corresponding CLI flags are `--runtimeAPI.*`; environment variables use
uppercase keys, e.g. `RUNTIMEAPI_INSTALLATIONUID`. The default is disabled and
does not require Kubernetes credentials. An enabled gateway fails startup if its
anchor is absent, terminating, or has a different UID. Preserve this configured
UID across restarts and upgrades.

When the native endpoint uses MongoDB, the gateway verifies the installation
before running schema migrations or serving traffic. On first startup with
`database.ddlAuto=true`, it records the configured UID in the unsharded
`runtime_metadata` collection using a majority write. With automatic DDL
disabled, provision this document in the gateway database once, using the live
anchor UID and a majority write concern:

```javascript
db.runtime_metadata.insertOne({ _id: "installation", uid: "<anchor UID>" }, { writeConcern: { w: "majority" } })
```

Back up this document with the Agent data. If a restored cluster has a different
anchor UID, startup stops even when the new UID matches the gateway configuration.
Recovery requires an explicit migration of installation identity. Recording the
installation does not enroll existing Agents or assign them to namespaces.

Grant the gateway ServiceAccount these native permissions:

| Scope | API group | Resource | Verb |
| --- | --- | --- | --- |
| ClusterRole | `authentication.k8s.io` | `tokenreviews` | `create` |
| ClusterRole | `authorization.k8s.io` | `subjectaccessreviews` | `create` |
| ClusterRole | core (`""`) | `namespaces` | `get` |
| Role in the installation namespace | core (`""`) | `configmaps`, restricted by `resourceNames` to the anchor | `get` |

Use the existing API TLS settings or a trusted TLS-terminating ingress for
bearer-token traffic. Do not forward kubeconfig client certificates or arbitrary
kubeconfig credentials to a Commander URL.

## Caller permission

The endpoint checks a virtual **namespaced** resource, not the core Kubernetes
Namespace resource and not an Agent CR. Bind this Role to the actual user, group,
or ServiceAccount in the target namespace:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: runtime-namespace-reader
  namespace: collectors
rules:
  - apiGroups: [runtime.opampcommander.io]
    resources: [namespaces]
    resourceNames: [collectors]
    verbs: [get]
```

For a machine client, issue a short-lived ServiceAccount TokenRequest with the
configured audience (`kubectl create token <service-account> --audience=opampcommander-runtime --duration=10m`).
Send it as `Authorization: Bearer <token>`. OIDC tokens are usable only if the
cluster's TokenReview authenticator accepts them for that audience; this endpoint
does not configure an OIDC provider or obtain user credentials.

Each request makes a fresh
[TokenReview](https://kubernetes.io/docs/reference/access-authn-authz/authentication/#webhook-token-authentication)
and [SubjectAccessReview](https://kubernetes.io/docs/reference/access-authn-authz/authorization/#checking-api-access).
Commander requires the requested audience in the authenticated result and sends
only its returned username, UID, groups and extra fields to authorization.
`X-Remote-User`, `X-Remote-Group` and `Impersonate-*` HTTP headers confer no identity.
The gateway's ServiceAccount is never substituted for a rejected caller.

Commander does not cache authentication, authorization or namespace ownership.
Revocation visibility still depends on the Kubernetes authenticators/authorizers
and their caches. All checks share a five-second request deadline. Invalid or
anonymous credentials return 401, denied permissions or namespace ownership
return 403, and review failures return 503 without backend details or tokens.
Missing namespaces return 404 only after caller authorization.

After authorization, the gateway rechecks the configured anchor and the target
namespace's `opampcommander.io/installation-uid` label. Anchor deletion/recreation
returns 503; unassigned, foreign or terminating namespaces are denied. Discovery
returns the current namespace UID: same-name recreation never reuses an old UID.
This response is a point-in-time lookup, not a durable authorization grant; future
Agent reads and mutations must validate enrolled UIDs and fencing independently.

## Verification and remaining cutover

Unit tests cover denied/anonymous/ambiguous credentials, audience enforcement,
forwarded identity forgery, exact SAR identity/scope, review failures and changing
ownership. The combined Kind suite runs **only in CI** and exercises real
TokenReview/SAR, RoleBinding removal, ServiceAccount recreation, namespace
recreation and installation-anchor replacement alongside the existing runtime
and Operator regressions.

Agent enrollment, scoped Agent operations and privileged payload permissions,
durable mutation audit, web/Grafana login and removal of legacy authentication
remain follow-up work under #902. No Agent mutation permission is introduced here.
