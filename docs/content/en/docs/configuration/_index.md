---
title: "Configuration"
linkTitle: "Configuration"
weight: 2
type: docs
description: >
  Configure the OpAMP Commander apiserver.
---

The apiserver is configured with a YAML file passed via `--config`, with individual
command-line flags, or with environment variables. A complete annotated example lives
at [`configs/apiserver/config.sample.yaml`](https://github.com/minuk-dev/opampcommander/blob/main/configs/apiserver/config.sample.yaml).

```bash
go run ./cmd/apiserver/main.go --config ./configs/apiserver/dev.yaml
```

## Precedence

Every YAML key has an equivalent dotted flag and environment variable. Command-line
flags override the config file. For example `management.log.level` can be set as:

```bash
--management.log.level=debug          # flag
MANAGEMENT_LOG_LEVEL=debug            # environment variable
```

## Server

```yaml
address: localhost:8080    # REST API + OpAMP WebSocket endpoint
serverId: ""               # defaults to hostname; also settable via SERVER_ID
serviceName: opampcommander
```

## Database

```yaml
database:
  type: "mongodb"          # "mongodb" or "inmemory" (inmemory for local/dev only)
  endpoints:
    - "mongodb://<user>:<password>@localhost:27017"
  connectTimeout: 10s
  databaseName: "opampcommander"
  ddlAuto: true            # create indexes/schema on startup
```

`inmemory` keeps no data across restarts and is intended for development and tests.

### Sharded cluster (MongoDB)

To scale horizontally beyond a single shard, point `endpoints` at your `mongos`
routers and turn on sharding-aware schema management. This is opt-in and requires
`ddlAuto: true`.

```yaml
database:
  type: mongodb
  endpoints:
    - "mongodb://<user>:<password>@mongos-0:27017,mongos-1:27017/?replicaSet="
  databaseName: opampcommander
  ddlAuto: true
  sharding:
    enabled: true          # enableSharding + shardCollection on startup
```

When `sharding.enabled` is set, startup additionally runs `enableSharding` on the
database and `shardCollection` for the collections that grow with fleet size. The
operation is idempotent — restarts against an already-sharded cluster are a no-op.

The shard-key plan is built into the server (it is tied to the collections' unique
indexes and query patterns, so it is not operator-configurable):

| Collection | Shard key | Rationale |
|---|---|---|
| `agents` | `{ "metadata.instanceUid": "hashed" }` | Largest collection; the unique index is on `metadata.instanceUid`, so the shard key must be prefixed by it. Hashed spreads writes evenly and keeps uniqueness single-shard; point lookups by instanceUid stay targeted, namespace list becomes scatter-gather. |
| `serverconnections` | `{ "uid": "hashed" }` | Grows with live agent connections; `uid` is unique by construction. |

All other collections (config-/cluster-scale: `agentgroups`, `agentpackages`,
`agentremoteconfigs`, `certificates`, `endpoints`, `namespaces`, `servers`,
`serverheartbeats`, `users`, and the RBAC collections) stay on the primary shard —
sharding them would add routing cost with no benefit. They are still created
explicitly at startup so they land deterministically. `hosts` and `containers` are
sharding candidates for very large fleets; they are created but not yet sharded.

## Events (single-node vs. multi-node)

```yaml
event:
  enabled: false           # false = standalone (single instance)
  type: "inmemory"         # "inmemory" for standalone, "kafka" for distributed
  kafka:
    brokers:
      - "localhost:9092"
    topic: "prod.opampcommander.events"
```

When running multiple apiserver instances, set `enabled: true` and `type: kafka` so a
management request received by one instance can be delivered to an agent connected to
another. See the protocol overview for the coordination flow.

## Agent liveness

Agent liveness — `connected`, `lastReportedAt`, `sequenceNum` — churns on every heartbeat
but is worthless once stale: the next message from the agent rebuilds it. The server
absorbs those updates in a fast tier and writes them through to the database on a slow
cadence, so a large fleet stops paying a database write per agent per heartbeat.

```yaml
liveness:
  flushInterval: 30s       # write-behind cadence
  flushStaleAfter: 30s     # how far behind a document must fall to be claimed
  flushBatchSize: 2000
  # persistThrottle: 10s   # leave unset unless you have a reason
  redis:
    enabled: false         # opt-in; absent or false is the default behaviour
    endpoints:
      - "redis:6379"
    dialTimeout: 2s
    commandTimeout: 200ms
    ttl: 120s
```

### What the settings mean

`flushInterval` and `flushStaleAfter` bound how stale the stored agent document may
get, and it is their **sum** that matters: a document ages by up to
`flushInterval + flushStaleAfter`, because the flush only claims documents already
`flushStaleAfter` behind and only looks every `flushInterval`. That sum has to stay
inside the **60s budget** the 90s connection-staleness window allows, and the server
refuses to start otherwise.

This is not just about surviving an outage. "Connected" is evaluated *inside the
database* in two places no read-side overlay can reach — the connected-agent list
filter and the agent-group connected counts — and both read the stored timestamp.
Let the document drift past the window and both start reporting live agents as
disconnected while their WebSockets are fine.

`persistThrottle` is the minimum interval between database writes on the *message*
path for an agent whose only change is that it is still alive. Leave it unset: 10s
with no shared fast tier (where it is the only thing keeping the document current),
and the 60s budget with one (where the write-behind flush owns the routine write
path). It is capped at the budget either way, for the same reason.

`ttl` must exceed the 90s staleness window, or a live agent's record would expire
while the agent is still considered connected. The server refuses to start otherwise.

`commandTimeout` is short on purpose. This is a fast path: a slow Redis must degrade
to the database rather than hold up agent message processing.

### Redis is optional, always

Redis is an accelerator and never a requirement. Absent, misconfigured, or down, the
server keeps working on node-local state and the database — just with more database
writes.

- A Redis that is **unreachable at startup** does not stop the server from starting.
- A Redis that **fails while running** trips a circuit breaker after a few consecutive
  failures. Calls route to node-local state, the server forces a write-behind flush so
  the database is current before reads start relying on it, and a probe every 30s
  routes back automatically once Redis returns. **No restart, no config change.**
- The outage shows up as **degraded** in `/healthz`, with the status code left at
  `200`: losing an optional accelerator must never be the reason a process is
  restarted. `/readyz` is unaffected.
- Redis is **ignored entirely in standalone mode** (`database.type: inmemory`), with a
  warning in the log. A standalone server keeps its whole state in process, so a shared
  tier would only add a network hop to reach data it already holds.

One endpoint addresses a single server, several address a cluster, and adding
`masterName` selects Sentinel.

### What it actually saves

The saving is not primarily *fewer* writes — the staleness window puts a floor under
how rarely the stored document can be refreshed. It is **much cheaper writes**: a
four-field `$set` instead of a full document rewrite, with no resource-version bump
and so no optimistic-concurrency churn against the reconcile loop and API writes.

Measured with `make bench-liveness`, per heartbeat per agent:

| Heartbeat | Path | Document rewrites | Liveness-only writes |
|---|---|---|---|
| 30s | every heartbeat (before this existed) | 1.00 | — |
| 30s | message-path throttle only | 0.50 | — |
| 30s | **write-behind** | **~0** | 1.00 |
| 5s | every heartbeat (before this existed) | 1.00 | — |
| 5s | message-path throttle only | 0.08 | — |
| 5s | **write-behind** | **~0** | 0.17 |

At the standard 30s OpAMP heartbeat, every full document rewrite becomes a narrow
`$set` — the same write *rate*, at a fraction of the cost, and with the version churn
gone. A chattier fleet wins on rate too: at a 5s heartbeat the write rate drops about
six-fold on top of that.

Note the middle rows: a throttle **shorter than the heartbeat interval buys nothing**,
because it has always elapsed by the time the next heartbeat lands. That is why the
throttle alone was never the answer.

### Metrics

| Metric | Meaning |
|---|---|
| `opampcommander_agent_liveness_absorbed` | Observations the fast tier took without a database write. |
| `opampcommander_agent_liveness_written` | Observations that reached the database, labelled `shape`: `document` (full rewrite) or `liveness` (narrow `$set`). |
| `opampcommander_agent_liveness_fallback` | Operations served by node-local state, by operation. |
| `opampcommander_agent_liveness_breaker_state` | 0 closed, 1 half-open, 2 open. |

`absorbed` minus `written` is the database writes the fast tier saved.

## Bootstrap (initial manifests)

On startup the server reconciles a directory of manifest YAML files into persistence
(declarative, full overwrite). The container image ships defaults at
`/etc/opampcommander/initial` (also exposed via the `BOOTSTRAP_DIR` env var).

```yaml
bootstrap:
  dir: /etc/opampcommander/initial
  remoteConfigSchemaLoad: latest   # latest | all | none
  defaultNamespace: default   # namespace for agents without a service.namespace
  defaultRole: default        # role auto-granted to every user
```

Setting `bootstrap.dir` empty disables bootstrapping.

### RemoteConfigSchema library

The image also ships a pre-built library of `RemoteConfigSchema` manifests under
`<bootstrap.dir>/remoteconfigschema`: one per OTel Collector release and
distribution, listing the components that release ships, the signals each one
handles, its stability, and the settings it accepts. On startup these are seeded
per `bootstrap.remoteConfigSchemaLoad`:

- `latest` (default) — seed only the newest version of each distribution
  (`otelcol`, `otelcol-contrib`, `otelcol-k8s`, `otelcol-otlp`)
- `all` — seed every version in the library
- `none` — seed nothing (the files remain on disk for opt-in use)

Point `bootstrap.remoteConfigSchemaDir` elsewhere to use a different library.

The library is rendered from the
[otelcol-config-schemas](https://github.com/minuk-dev/otelcol-config-schemas)
registry, whose files are generated by `tools/schemagen` in
[otelcol-config-lint](https://github.com/minuk-dev/otelcol-config-lint) from each
release's upstream `metadata.yaml`, its components' `Config` structs, and the
`config.schema.yaml` the collector publishes from v0.150.0 on. The linter reads the
same registry, so a config checked locally with `otelcol-config-lint` is checked
against the same catalog the server holds.

Regenerate the library with `make gen-remoteconfigschema`, or pull a single release
yourself:

```sh
opampctl generate remoteconfigschema --distribution contrib | opampctl create -f -
opampctl generate remoteconfigschema --distribution core --version v0.150.0 > schema.yaml
```

Releases the registry does not cover are handled by the collector itself: point
`--binary-path` at any distribution, including a custom one, and its `components`
output becomes a schema. Such a schema lists the components the binary has but not
the settings they accept, so configs targeting it are checked for component
existence and signal support only.

```sh
opampctl generate remoteconfigschema --binary-path ./otelcol-custom > schema.yaml
```

## Management (observability)

The management server runs on a separate address and hosts health checks, metrics,
pprof, logging, and tracing configuration.

```yaml
management:
  address: localhost:9090
  metric:
    enabled: true
    type: prometheus              # "prometheus" or "opentelemetry"
    prometheus:
      path: /metrics
    opentelemetry:                # used when type is opentelemetry
      endpoint: "localhost:4317"
  log:
    level: "info"                 # debug, info, warn, error
    format: "json"                # json or text
  trace:
    enabled: false
    endpoint: "localhost:4317"
    protocol: grpc                # grpc, http/protobuf, http/json
    sampler: always               # always, never, probability
```

Health checks are served at `GET /healthz` and `GET /readyz`.

## Authentication

OpAMP Commander supports OAuth2 (GitHub), basic auth (with hashed passwords), and a
manual bearer-token mode used by the CLI.

```yaml
auth:
  enabled: true
  admin:
    username: "admin"
    password: "admin_password"
    email: "admin@admin"
  basic:
    # Server-side secret mixed into every basic-auth password hash. Set a long,
    # random, stable value. Empty disables DB-backed basic-auth users.
    pepper: ""
  jwt:
    issuer: "opampcommander"
    expire: 30m                   # access token lifetime
    refreshExpire: 168h           # refresh token lifetime (0 disables refresh)
    secret: "your_jwt_secret"
    audience:
      - "opampcommander"
  type: "oauth2"
  oauth2:
    provider: github
    clientId: "your_client_id"
    clientSecret: "your_client_secret"
    redirectUri: "http://localhost:8080/auth/callback"
    # Extra hosts the authcode endpoint accepts as redirect targets, on top of the
    # always-allowed loopback hosts (127.0.0.1, ::1, localhost). Add your web UI host.
    allowedRedirectHosts:
      - opampcommander.minuk.dev
    state:                        # CSRF protection for the OAuth2 flow
      mode: jwt
      jwt:
        issuer: "opampcommander"
        expire: 5m
        secret: "your_jwt_secret"
        audience:
          - "opampcommander"
```

With `auth.enabled: false`, authentication is bypassed — only suitable for local
development.

## Command-line flags

Every option above has a flag. A few common ones:

| Flag | Default | Description |
|---|---|---|
| `--config` | — | Path to the YAML config file |
| `--address` | `localhost:8080` | API + OpAMP WebSocket address |
| `--database.type` | `inmemory` | `inmemory` or `mongodb` |
| `--database.endpoints` | `mongodb://localhost:27017` | Database endpoints |
| `--database.sharding.enabled` | `false` | Sharding-aware schema management (needs `--database.ddlAuto`) |
| `--event.enabled` | `false` | Enable multi-node events |
| `--event.type` | `inmemory` | `inmemory` or `kafka` |
| `--management.address` | `localhost:9090` | Management server address |
| `--management.log.level` | `info` | Log level |
| `--auth.enabled` | `false` | Enable authentication |

Run `apiserver --help` for the complete list.
