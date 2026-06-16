---
title: "Getting Started"
linkTitle: "Getting Started"
weight: 1
type: docs
description: >
  Install OpAMP Commander, run the apiserver, and connect your first agent.
---

## What you'll set up

OpAMP Commander has three components:

- **apiserver** — the server agents connect to and that exposes the management REST API.
- **opampctl** — the command-line client.
- **web** — an optional dashboard.

This guide gets the apiserver running and `opampctl` talking to it.

## Prerequisites

- Go 1.25 or later
- Docker (to run MongoDB, and Kafka for multi-node mode)
- Node.js 20.9+ (only if you want the web dashboard)

## Get the source

```bash
git clone https://github.com/minuk-dev/opampcommander.git
cd opampcommander
```

## Run the apiserver

The Makefile wraps the common workflows.

### Standalone mode (single node)

Runs against MongoDB only, with an in-memory event bus — no Kafka required:

```bash
make run-standalone
```

### Development mode (with Kafka)

Starts MongoDB **and** Kafka in Docker, then runs the server:

```bash
make run-dev-server
```

### Run directly

You can also run the binary against a config file:

```bash
go run ./cmd/apiserver/main.go --config ./configs/apiserver/dev.yaml
```

By default the server listens on:

- `localhost:8080` — REST API and the OpAMP WebSocket endpoint (`/api/v1/opamp`)
- `localhost:9090` — management endpoints (`/healthz`, `/readyz`, metrics, pprof)

See the [Configuration guide](/en/docs/configuration/) for every available option.

## Install opampctl

```bash
go install github.com/minuk-dev/opampcommander/cmd/opampctl@latest
```

Or build from the repository:

```bash
go build -o opampctl ./cmd/opampctl
```

Create a configuration file and verify the connection:

```bash
opampctl config init     # writes ~/.config/opampcommander/opampctl/config.yaml
opampctl whoami
opampctl get agent
```

See the [CLI reference](/en/docs/cli/) for the full command set.

## Connect an agent

Point any OpAMP-capable OpenTelemetry Collector at the server's WebSocket endpoint:

```yaml
extensions:
  opamp:
    server:
      ws:
        endpoint: ws://localhost:8080/api/v1/opamp
```

The agent's `service.namespace` resource attribute determines its namespace
(defaulting to `default`).

## Next steps

- Configure the server in the [Configuration guide](/en/docs/configuration/)
- Explore the [REST API](/en/docs/api/)
- Learn the [CLI commands](/en/docs/cli/)
