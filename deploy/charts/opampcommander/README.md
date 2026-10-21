# opampcommander Helm chart

Deploys the [OpAMP Commander](https://github.com/minuk-dev/opampcommander)
apiserver — the server OpenTelemetry collectors connect to over OpAMP — and,
optionally, the Next.js web dashboard.

## Quick start

```sh
make helm-deps
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

## Optional Grafana preview

Grafana is disabled by default. Enabling it uses the pinned Grafana community
subchart and requires Kubernetes >=1.25. It provisions the Commander app and
preview data source in organization 1. **Configure an upstream URL, namespace and access token** before querying.
The default data source is unconfigured and reports a configuration error.

Build a Grafana image containing the plugin before enabling it:

```sh
cd plugins/grafana
npm ci
make image IMAGE=opampcommander-grafana:dev
cd ../..
# For a local kind cluster: kind load docker-image opampcommander-grafana:dev --name <cluster>
# For a remote cluster: tag/push to your registry and override grafana.image below.
make helm-deps
helm upgrade --install commander ./deploy/charts/opampcommander \
  --namespace commander --create-namespace \
  -f deploy/charts/opampcommander/ci/grafana-values.yaml
kubectl -n commander port-forward svc/commander-grafana 3300:80
```

Open `http://127.0.0.1:3300/a/minukdev-opampcommander-app`. Sign in as `admin`;
read the generated password with:

```sh
kubectl -n commander get secret commander-grafana \
  -o jsonpath='{.data.admin-password}' | base64 -d; echo
```

The example explicitly allows the two unsigned preview plugin IDs. Use signed
plugins for production; signing and release publishing are tracked in #809.
There is no published Commander Grafana image yet. `make image` builds for the
local Docker architecture; build/push the image for your cluster's architecture.
Custom images must retain the bundled plugin at `/opt/opampcommander/plugins` so
Grafana's data volume cannot hide it. No plugins are downloaded at startup.

Configure the existing data-source entry and inject a token from a Secret:

```yaml
grafana:
  envValueFrom:
    COMMANDER_ACCESS_TOKEN:
      secretKeyRef:
        name: commander-grafana-access
        key: token
  datasources:
    opampcommander.yaml:
      apiVersion: 1
      datasources:
        - name: OpAMP Commander
          uid: opampcommander-preview
          type: minukdev-opampcommander-datasource
          access: proxy
          orgId: 1
          editable: false
          jsonData:
            url: https://commander.example.com
            namespace: default
          secureJsonData:
            token: $COMMANDER_ACCESS_TOKEN
```

The Secret must exist in the Grafana namespace. Its token needs `agent:LIST` in
the configured namespace. Restart Grafana after rotating the Secret so provisioning
reads the replacement token. Automated machine-token renewal remains in #806.

All `grafana.*` values pass through to the [community chart](https://github.com/grafana-community/helm-charts/tree/main/charts/grafana).
Set `grafana.image.registry`, `repository` and `tag` for your registry. Anonymous
access, ingress and persistence are disabled by default; no cluster RBAC or
service-account token is required. Use `grafana.admin.existingSecret` (keys
`admin-user` and `admin-password`) for GitOps installs. Enable
`grafana.persistence.enabled` to retain dashboards/settings and choose a storage
class that supports the Grafana pod's fsGroup 472. The Grafana image version is
independent of the Commander chart's appVersion.

`grafana.extraConfigmapMounts` contains the app provisioning mount; preserve that
entry when adding mounts. Provisioning files are read at startup, so restart the
Grafana deployment after changing them. To use an existing Grafana installation,
leave `grafana.enabled=false` and install the plugin there separately.

## A real Commander install

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

### Mounted secret files

To read credentials from a Secret volume instead of injecting their values as
pod environment variables, use [ci/secret-files-values.yaml](ci/secret-files-values.yaml):

```sh
helm upgrade --install opampcommander ./deploy/charts/opampcommander \
  -f ./deploy/charts/opampcommander/ci/secret-files-values.yaml
```

Create the referenced `opampcommander-secret-files` Secret in the release
namespace first, with every key named in the example. `databaseURI` holds one
complete MongoDB URI, including any replica-set host list. The example reuses
`extraEnv`, `extraVolumes`, and `extraVolumeMounts`; credentials never enter the
chart ConfigMap. Set the appropriate public OAuth2 client ID and callback URL
separately. Mount files readable by the pod's configured user/group.

Keep `secrets.create: false` and `secrets.existingSecret: ""` in file mode so the
chart does not inject direct secret environment values, which would take
precedence over `_FILE`. A missing or empty selected file prevents startup.
Restart the deployment after changing mounted secrets; values are read at startup.

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

Health, metrics and pprof are served on a second port, which the chart puts on
its own `<release>-apiserver-management` Service, **always ClusterIP**. It is
separate from the API Service on purpose: Kubernetes publishes every port of a
NodePort or LoadBalancer Service externally, so sharing one Service would expose
pprof on a node port — or on the internet — the moment `apiserver.service.type`
changed.

`apiserver.serviceMonitor.enabled` creates a Prometheus Operator ServiceMonitor
against that Service, using the path from
`apiserver.config.management.metric.prometheus.path`.

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
