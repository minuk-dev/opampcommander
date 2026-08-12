# opampcommander Helm chart

Deploys the [OpAMP Commander](https://github.com/minuk-dev/opampcommander)
apiserver — the server OpenTelemetry collectors connect to over OpAMP — and,
optionally, the Next.js web dashboard.

## Quick start

```sh
helm install opampcommander ./deploy/charts/opampcommander \
  --namespace opampcommander --create-namespace
```

The defaults install a **self-contained demo**: one apiserver replica backed by
the in-memory store, plus the dashboard. Nothing outside the cluster is needed
and nothing survives a restart.

Reach it, and read the generated admin password:

```sh
kubectl -n opampcommander port-forward svc/opampcommander-web 3000:3000
kubectl -n opampcommander get secret opampcommander-apiserver \
  -o jsonpath='{.data.adminPassword}' | base64 -d; echo
```

Verify a release with the bundled connectivity test:

```sh
helm test opampcommander --namespace opampcommander
```

## A real install

Two things must change for anything beyond a demo: persistence and secrets.

```yaml
# values.yaml
apiserver:
  replicaCount: 3
  config:
    database:
      type: mongodb
      # A replica set is required — the server uses transactions.
      endpoints:
        - "mongodb://mongodb-0.mongodb:27017,mongodb-1.mongodb:27017/?replicaSet=rs0"
      databaseName: opampcommander
      ddlAuto: true
    event:
      # Required with more than one replica, so a management request handled by
      # one replica reaches an agent connected to another.
      type: kafka
      kafka:
        brokers: ["kafka-0.kafka:9092"]
        topic: "prod.opampcommander.events"
    auth:
      oauth2:
        redirectUri: "https://opampcommander.example.com/auth/callback"
        allowedRedirectHosts: ["opampcommander.example.com"]
  secrets:
    create: false
    existingSecret: opampcommander-credentials
  ingress:
    enabled: true
    className: nginx
    annotations:
      # The OpAMP endpoint is a long-lived WebSocket.
      nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
      nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
    hosts:
      - host: api.opampcommander.example.com
        paths: [{ path: /, pathType: Prefix }]

web:
  ingress:
    enabled: true
    className: nginx
    hosts:
      - host: opampcommander.example.com
        paths: [{ path: /, pathType: Prefix }]
```

The chart refuses to render combinations that cannot work — more than one
replica on the in-memory store or the in-memory event bus, `type: mongodb` with
no endpoints, `type: kafka` with no brokers — rather than installing something
subtly broken.

Ready-made permutations live in [`ci/`](ci/) and are rendered on every CI run.

### Chart does not deploy MongoDB or Kafka

Both are stateful, opinionated, and usually already operated elsewhere, so the
chart only points at them. Install them separately (an operator, a cloud
service, or a dependency chart of your own) and set the endpoints above.

## Configuration

`apiserver.config` is rendered verbatim into a ConfigMap and mounted at
`/etc/opampcommander/config/config.yaml`. Every key documented in the
[configuration guide](https://minuk-dev.github.io/opampcommander/docs/configuration/) is
accepted there; the chart does not maintain a parallel schema.

Two things are handled outside that map:

- **Credentials** come from a Secret as environment variables, which the server
  reads with a higher precedence than the config file. Never put them in
  `apiserver.config` — a ConfigMap is world-readable to anything in the
  namespace.
- **Ports** are derived from `apiserver.config.address` and
  `apiserver.config.management.address`, so the container ports, probes and
  Service can never drift from what the process actually binds. Both must bind
  `0.0.0.0`, not `localhost`, or the pod is unreachable.

### Secrets

| Secret key | Environment variable | Purpose |
|---|---|---|
| `adminUsername` | `AUTH_ADMIN_USERNAME` | Built-in admin login |
| `adminPassword` | `AUTH_ADMIN_PASSWORD` | Built-in admin login |
| `jwtSecret` | `AUTH_JWT_SECRET` | Signs issued access tokens |
| `basicPepper` | `AUTH_BASIC_PEPPER` | Peppers stored basic-auth password hashes |
| `oauth2ClientId` | `AUTH_OAUTH2_CLIENTID` | GitHub OAuth2 app |
| `oauth2ClientSecret` | `AUTH_OAUTH2_CLIENTSECRET` | GitHub OAuth2 app |
| `oauth2StateJwtSecret` | `AUTH_OAUTH2_STATE_JWT_SECRET` | Signs the OAuth2 CSRF state token |
| `directAuthToken` | `EVENT_DIRECT_AUTHTOKEN` | Pre-shared token between peers in `event.type: direct` |

`adminPassword`, `jwtSecret`, `basicPepper` and `oauth2StateJwtSecret` are
generated when left empty and carried forward on upgrade by reading the Secret
already in the cluster.

> That lookup returns nothing under `helm template` and `--dry-run`. A
> render-then-apply (GitOps/Argo CD) flow therefore **must** set them
> explicitly, or every render rotates them — invalidating every issued token and
> every stored password. Point `apiserver.secrets.existingSecret` at a Secret
> you manage (External Secrets, Sealed Secrets, …) and set
> `apiserver.secrets.create: false`.

### Multi-replica coordination

| `event.type` | What it needs | Notes |
|---|---|---|
| `inmemory` | nothing | Single replica only |
| `kafka` | a broker | Events are broadcast; every replica filters |
| `direct` | nothing | Peers dial each other; exactly one replica receives |

In `direct` mode the chart opens `apiserver.containerPorts.direct` (8081) and
advertises `$(POD_IP):8081` from the downward API, so peers can resolve each
other through the server registry with no extra configuration. Set
`apiserver.secrets.directAuthToken` unless the cluster network is trusted.

## Monitoring

`apiserver.serviceMonitor.enabled` creates a Prometheus Operator ServiceMonitor
against the management port, using the path from
`apiserver.config.management.metric.prometheus.path`. The management port is
deliberately kept off the ingress — it also serves pprof.

## Values

See [`values.yaml`](values.yaml); every key is commented. The most commonly
changed ones:

| Key | Default | Description |
|---|---|---|
| `apiserver.replicaCount` | `1` | apiserver replicas |
| `apiserver.image.tag` | `""` | Defaults to the chart's appVersion |
| `apiserver.config` | see values | Contents of the apiserver `config.yaml` |
| `apiserver.existingConfigMap` | `""` | Use a ConfigMap you manage instead |
| `apiserver.secrets.existingSecret` | `""` | Use a Secret you manage instead |
| `apiserver.ingress.enabled` | `false` | Expose the REST API and OpAMP endpoint |
| `apiserver.serviceMonitor.enabled` | `false` | Prometheus Operator scrape config |
| `apiserver.autoscaling.enabled` | `false` | HPA for the apiserver |
| `web.enabled` | `true` | Deploy the dashboard |
| `web.apiUrl` | `""` | Defaults to this release's apiserver Service |
| `web.ingress.enabled` | `false` | Expose the dashboard |

## Development

```sh
make helm-lint       # helm lint across every ci/ permutation
make helm-template    # render each permutation to stdout
```
