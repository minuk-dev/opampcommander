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

## Bootstrap (initial manifests)

On startup the server reconciles a directory of manifest YAML files into persistence
(declarative, full overwrite). The container image ships defaults at
`/etc/opampcommander/initial` (also exposed via the `BOOTSTRAP_DIR` env var).

```yaml
bootstrap:
  dir: /etc/opampcommander/initial
  defaultNamespace: default   # namespace for agents without a service.namespace
  defaultRole: default        # role auto-granted to every user
```

Setting `bootstrap.dir` empty disables bootstrapping.

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
| `--event.enabled` | `false` | Enable multi-node events |
| `--event.type` | `inmemory` | `inmemory` or `kafka` |
| `--management.address` | `localhost:9090` | Management server address |
| `--management.log.level` | `info` | Log level |
| `--auth.enabled` | `false` | Enable authentication |

Run `apiserver --help` for the complete list.
