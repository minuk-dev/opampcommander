---
title: "Deployment"
linkTitle: "Deployment"
weight: 6
type: docs
description: >
  Deploy the apiserver for OpAMP and connect it to the Kubernetes control plane.
---

The Helm chart in
[`deploy/charts/opampcommander`](https://github.com/minuk-dev/opampcommander/tree/main/deploy/charts/opampcommander)
installs the apiserver that collectors connect to over OpAMP.

## Kubernetes topology

Install the Operator and its CRDs for user-facing management. Kubernetes
enforces caller permissions. The apiserver runs the OpAMP endpoint and internal
REST. The Helm chart keeps the REST Service private and routes only the Agent
OpAMP path through its optional Ingress.

### WebSocket ingress requirements

Preserve the HTTP/1.1 `Upgrade: websocket` and `Connection: upgrade` handshake
on `/api/v1/opamp`. Set proxy read/send and load-balancer idle timeouts longer
than the agent heartbeat/ping interval, with room for delayed heartbeats
(the example below uses 3600 seconds). Apply compatible timeouts at every hop,
not only the Kubernetes ingress, and allow the expected message body sizes.

An established WebSocket stays on one backend connection. Cookie affinity is
not required for it, and reconnects may choose another replica. Affinity is
not a replacement for shared MongoDB and a cross-server event backend; HTTP
polling and management requests can reach different replicas. Keep management
port 9090 private. For agent mTLS, follow the TLS passthrough requirements below.

### Image configuration reference

Apiserver image:

| Configuration | Environment equivalent | Purpose |
|---|---|---|
| `--config /path/config.yaml` | — | Mounted YAML configuration file |
| `address` | `ADDRESS` | API/OpAMP listener; use `0.0.0.0:8080` in a container |
| `management.address` | `MANAGEMENT_ADDRESS` | Private health/metrics/pprof listener |
| `database.type`, `database.endpoints` | `DATABASE_TYPE`, `DATABASE_ENDPOINTS` | `inmemory` demo or MongoDB; endpoints can be comma-separated |
| `event.type`, `event.kafka.brokers` | `EVENT_TYPE`, `EVENT_KAFKA_BROKERS` | Single-node `inmemory` or distributed event backend |
| `bootstrap.dir` | `BOOTSTRAP_DIR` | Image sets `/etc/opampcommander/initial` |

All registered configuration keys follow the same rule: uppercase and replace
`.` with `_` (for example `database.databaseName` → `DATABASE_DATABASENAME`).
Secret fields also accept `_FILE` paths; precedence is explicit flags, nonempty
environment values, secret files, YAML, defaults (see
[External secrets](/docs/configuration/#external-secrets)). Use
`/apiserver config view --config /path/config.yaml` to inspect effective settings
with secrets redacted, and `/apiserver --help` for the full flag list.
See [Configuration](/docs/configuration/) for TLS and other settings.

## Local Compose quickstart

From the repository root, set `OPAMP_VERSION` to an apiserver image tag
that includes the private REST API changes, then start the loopback demo:

```sh
docker compose -f deploy/compose.yaml up -d
```

Agents on the host use `ws://localhost:8080/api/v1/opamp`. The demo stores
state in memory. Stop it with `docker compose -f deploy/compose.yaml down`.

## Install

From a checkout of the repository:

```bash
helm install opampcommander ./deploy/charts/opampcommander \
  --namespace opampcommander --create-namespace
```

Or from the published OCI chart:

```bash
helm install opampcommander oci://ghcr.io/minuk-dev/charts/opampcommander \
  --namespace opampcommander --create-namespace
```

The defaults install one apiserver replica with an in-memory store.

Check a release end to end with the bundled connectivity test:

```bash
helm test opampcommander --namespace opampcommander
```

## Going to production

Production deployments need persistence, stable secrets, and TLS for agent and API traffic.

### Persistence

The in-memory store is per-process. Point the chart at MongoDB — a replica set,
because the server uses transactions:

```yaml
apiserver:
  config:
    database:
      type: mongodb
      endpoints:
        - "mongodb://mongodb-0.mongodb:27017,mongodb-1.mongodb:27017/?replicaSet=rs0"
      databaseName: opampcommander
      ddlAuto: true
```

The chart does not deploy MongoDB or Kafka. Both are stateful and usually
already operated elsewhere, so the chart only points at them.

### Multiple replicas

With more than one replica, a management request may arrive at a replica that
does not hold the target agent's WebSocket. `event.type` selects how that
request is forwarded:

| `event.type` | Requires | Behaviour |
|---|---|---|
| `inmemory` | nothing | Single replica only |
| `kafka` | a broker | Events are broadcast; every replica filters |
| `direct` | nothing | Peers dial each other; exactly one replica receives |

```yaml
apiserver:
  replicaCount: 3
  config:
    event:
      type: kafka
      kafka:
        brokers: ["kafka-0.kafka:9092"]
        topic: "prod.opampcommander.events"
```

In `direct` mode the chart opens the peer port and advertises `$(POD_IP):8081`
from the downward API, so replicas find each other through the server registry
with no extra configuration.

The chart refuses to render combinations that cannot work — several replicas on
the in-memory store or the in-memory event bus, `mongodb` with no endpoints,
`kafka` with no brokers — instead of installing something subtly broken.

### Secrets

Use mounted Secrets or environment variables for MongoDB and Redis credentials.
For direct events, `apiserver.secrets.existingSecret` can provide the
`directAuthToken` key. See [Configuration](/docs/configuration/) for file-backed
credentials.

## Exposing the server

```yaml
apiserver:
  ingress:
    enabled: true
    className: nginx
    annotations:
      # /api/v1/opamp is a long-lived WebSocket upgrade.
      nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
      nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
    tls:
      - secretName: opampcommander-api-tls
        hosts: [api.opampcommander.example.com]
    hosts:
      - host: api.opampcommander.example.com
```

The chart Ingress routes only `/api/v1/opamp`.

### TLS at the ingress

A supported topology is `Agent → HTTPS/WSS → ingress → HTTP/WS → apiserver`
with a private backend network. Provision the ingress TLS Secret,
redirect or reject external plaintext HTTP, and allow backend API traffic only
from the trusted ingress (for example with Kubernetes NetworkPolicies). The
apiserver can leave `tls` unset in this topology; its plaintext startup warning
is expected. Preserve WebSocket Upgrade/Connection headers and use long proxy
timeouts for `/api/v1/opamp`. Agents connect to
`wss://api.opampcommander.example.com/api/v1/opamp`.

The API currently uses Gin's default client-IP handling: `X-Forwarded-For` and
`X-Real-IP` are accepted from any peer for client-IP logging. Restrict direct
backend access and configure the ingress to discard untrusted incoming values
and set these headers itself. `Forwarded` and `X-Forwarded-Proto` do not establish
TLS or authenticated identity in the apiserver; PROXY protocol is not supported.
OpAMP mTLS checks the Agent certificate on the actual TLS connection.
Use TLS passthrough to the apiserver for OpAMP mTLS so the agent certificate
reaches the listener, rather than terminating that connection at the ingress.

### Upgrade from a release with Commander login

Remove the old `auth` section, `AUTH_*` settings, and User or Role manifests
from custom bootstrap directories. Route user management through the
Kubernetes API. Helm defaults leave the legacy dashboard disabled. If your
existing release explicitly enables it, provide a separate legacy backend
or set `web.enabled: false`.

### TLS at the apiserver

For native HTTPS/WSS, set `apiserver.config.tls` as described in the
[Configuration guide](/docs/configuration/#listener-tls). Mount a TLS Secret
using the chart's existing `apiserver.extraVolumes` and
`apiserver.extraVolumeMounts`; avoid `subPath` mounts if the files must rotate.
Configure ingress upstream HTTPS or TLS passthrough when using this listener.
Send SIGHUP to each apiserver process after the mounted certificate and key
have updated.

If enabling `management.tls`, set `httpGet.scheme: HTTPS` on the chart's
liveness and readiness probes (and startup probe when configured). Kubernetes
HTTP probes cannot present client certificates: with `management.tls.caFile`,
supply suitable authenticated exec probes instead. Configure Prometheus to use
HTTPS, the server CA, and client credentials when required; the bundled
ServiceMonitor uses HTTP, so disable it and provide your own HTTPS
ServiceMonitor for this topology.

## Grafana migration

Use the [standalone web migration checklist](../grafana-migration/) to map
existing workflows and rehearse retiring the standalone service.

## Configuration

`apiserver.config` is rendered verbatim into a ConfigMap and mounted at
`/etc/opampcommander/config/config.yaml`, so every key in the
[Configuration guide](/docs/configuration/) is accepted as-is — the chart keeps
no parallel schema. Container ports, probes and the Service are derived from
`address` and `management.address`, which must bind `0.0.0.0` rather than
`localhost` for the pod to be reachable.

See the
[chart README](https://github.com/minuk-dev/opampcommander/blob/main/deploy/charts/opampcommander/README.md)
for the full values reference, and
[`ci/`](https://github.com/minuk-dev/opampcommander/tree/main/deploy/charts/opampcommander/ci)
for complete example values files.

## Connect an agent

Point a collector's OpAMP extension at the apiserver's WebSocket endpoint:

```yaml
extensions:
  opamp:
    server:
      ws:
        endpoint: ws://opampcommander-apiserver.opampcommander.svc:8080/api/v1/opamp
```

## Uninstall

```bash
helm uninstall opampcommander --namespace opampcommander
```

The generated credentials Secret is removed with the release. Data in MongoDB is
not touched.
