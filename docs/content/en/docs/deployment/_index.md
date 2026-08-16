---
title: "Deployment"
linkTitle: "Deployment"
weight: 6
type: docs
description: >
  Run OpAMP Commander on Kubernetes with the Helm chart.
---

The Helm chart in
[`deploy/charts/opampcommander`](https://github.com/minuk-dev/opampcommander/tree/main/deploy/charts/opampcommander)
installs the apiserver — the server collectors connect to over OpAMP — and,
optionally, the web dashboard.

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

The defaults install a **self-contained demo**: one apiserver replica backed by
the in-memory store, plus the dashboard. Nothing outside the cluster is
required, and nothing survives a restart.

Read the generated admin password and open the dashboard:

```bash
kubectl -n opampcommander get secret opampcommander-apiserver \
  -o jsonpath='{.data.adminPassword}' | base64 -d; echo
kubectl -n opampcommander port-forward svc/opampcommander-web 3000:3000
```

Check a release end to end with the bundled connectivity test:

```bash
helm test opampcommander --namespace opampcommander
```

## Going to production

Two defaults must change: persistence and secrets.

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

Credentials are not part of `apiserver.config`; they live in a Secret and are
injected as environment variables, which the server reads with a higher
precedence than the config file.

| Secret key | Environment variable | Purpose |
|---|---|---|
| `adminUsername` | `AUTH_ADMIN_USERNAME` | Built-in admin login |
| `adminPassword` | `AUTH_ADMIN_PASSWORD` | Built-in admin login |
| `jwtSecret` | `AUTH_JWT_SECRET` | Signs issued access tokens |
| `basicPepper` | `AUTH_BASIC_PEPPER` | Peppers stored basic-auth password hashes |
| `oauth2ClientId` | `AUTH_OAUTH2_CLIENTID` | GitHub OAuth2 app |
| `oauth2ClientSecret` | `AUTH_OAUTH2_CLIENTSECRET` | GitHub OAuth2 app |
| `oauth2StateJwtSecret` | `AUTH_OAUTH2_STATE_JWT_SECRET` | Signs the OAuth2 CSRF state token |
| `directAuthToken` | `EVENT_DIRECT_AUTHTOKEN` | Pre-shared token between peers in `direct` mode |

Values left empty are generated on install and carried forward on upgrade by
reading the Secret already in the cluster.

{{% alert title="GitOps" color="warning" %}}
That lookup returns nothing under `helm template` and `--dry-run`. A
render-then-apply flow (Argo CD, Flux) must supply the secrets itself, or every
render rotates them — invalidating every issued token and every stored
password. Set `apiserver.secrets.create: false` and point
`apiserver.secrets.existingSecret` at a Secret you manage.
{{% /alert %}}

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
    hosts:
      - host: api.opampcommander.example.com
        paths:
          - path: /
            pathType: Prefix

web:
  ingress:
    enabled: true
    className: nginx
    hosts:
      - host: opampcommander.example.com
        paths:
          - path: /
            pathType: Prefix
```

For OAuth2 logins from the dashboard, set the callback and the allowed redirect
host to match:

```yaml
apiserver:
  config:
    auth:
      oauth2:
        redirectUri: "https://opampcommander.example.com/auth/callback"
        allowedRedirectHosts: ["opampcommander.example.com"]
```

Health, metrics and pprof are served on a second port, which the chart puts on
its own `<release>-apiserver-management` Service, always ClusterIP — separate
from the API Service so that switching `apiserver.service.type` to NodePort or
LoadBalancer cannot publish pprof externally. Scrape it with the Prometheus
Operator instead:

```yaml
apiserver:
  serviceMonitor:
    enabled: true
```

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
