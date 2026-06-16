---
title: "CLI Reference"
linkTitle: "CLI"
weight: 4
type: docs
description: >
  Use the opampctl command-line tool to manage OpAMP Commander.
---

`opampctl` is a `kubectl`-style command-line client for the OpAMP Commander apiserver.
It manages agents, agent groups, packages, remote configs, certificates, namespaces,
users, roles, and role bindings.

## Installation

```bash
go install github.com/minuk-dev/opampcommander/cmd/opampctl@latest
```

Or build from source:

```bash
git clone https://github.com/minuk-dev/opampcommander.git
cd opampcommander
go build -o opampctl ./cmd/opampctl
```

## Configuration

Initialize a configuration file:

```bash
opampctl config init
```

This creates `~/.config/opampcommander/opampctl/config.yaml`. The format follows the
`kubectl` model of **contexts**, **clusters**, and **users**:

```yaml
currentContext: default-1
contexts:
  - name: default-1
    cluster: default
    user: admin
  - name: default-2
    cluster: default
    user: githubuser
  - name: default-3
    cluster: default
    user: manual
users:
  - name: admin
    auth:
      type: basic
      username: admin
      password: admin
  - name: githubuser
    auth:
      type: github          # browser or device OAuth flow
  - name: manual
    auth:
      type: manual
      bearerToken: "<your-bearer-token>"
clusters:
  - name: default
    opampcommander:
      endpoint: http://localhost:8080
```

Three authentication types are supported per user:

- `basic` — username/password basic auth
- `github` — GitHub OAuth (browser or device flow; see `--auth-flow`)
- `manual` — a bearer token you supply directly

View the active configuration:

```bash
opampctl config view
```

## Global flags

These flags apply to every command:

| Flag | Default | Description |
|---|---|---|
| `-c`, `--config` | `~/.config/opampcommander/opampctl/config.yaml` | Path to the config file |
| `-v`, `--verbose` | `false` | Verbose output (equivalent to `--log.level=debug`) |
| `--log.format` | `text` | Log format (`text`, `json`) |
| `--log.level` | `info` | Log level (`debug`, `info`, `warn`) |
| `--auth-flow` | — | Override the GitHub auth flow: `browser` or `device` |

## Command overview

| Command | Description |
|---|---|
| `get` | List or read resources |
| `create` | Create resources (often from a `-f` manifest file) |
| `set` | Update fields of a resource |
| `delete` | Delete resources |
| `template` | Print example manifests (`template examples ...`) |
| `restart` | Send a restart command to agents |
| `config` | Manage the local config file (`init`, `view`) |
| `context` | Switch between contexts (`ls`, `use`) |
| `whoami` | Show the authenticated identity |
| `version` | Show client version |

### Resource types

`get`, `create`, and `delete` operate on these resource types:

`agent`, `agentgroup`, `agentpackage`, `agentremoteconfig`, `certificate`,
`connection` (get only), `container` (get only), `host` (get only), `namespace`,
`user`, `role`, `rolebinding`.

## Contexts

```bash
opampctl context ls              # list contexts, highlighting the current one
opampctl context use default-2   # switch context
```

## Authentication status

```bash
opampctl whoami
```

## Agents

```bash
# list agents in the default namespace
opampctl get agent

# output formats: short (default), text, json, yaml
opampctl get agent -o yaml

# filter by agent group
opampctl get agent --agentgroup my-group -n my-namespace

# across all namespaces
opampctl get agent -A
```

Update an agent (for example, assign a new instance UID):

```bash
opampctl set agent <instance-uid> --new-instance-uid <new-uid> -n default
```

Restart agents:

```bash
opampctl restart agent <instance-uid>
```

## Agent groups

```bash
opampctl get agentgroup
opampctl get agentgroup <name>

# create from a manifest file
opampctl create agentgroup -f ./agentgroup.yaml

# print an example manifest to start from
opampctl template examples agentgroup

opampctl delete agentgroup <name>
```

Most `create` commands accept `-f/--file` pointing at a YAML manifest. Use
`opampctl template examples <resource>` to print a starting template for a resource
type (`agentgroup`, `agentpackage`, `agentremoteconfig`, `certificate`, `namespace`,
`role`, `rolebinding`).

## Connections

```bash
opampctl get connection      # active agent connections
```

## Users and RBAC

```bash
# create a user, optionally with a basic-auth password
opampctl create user --username alice --email alice@example.com --password-stdin

opampctl get user
opampctl get role
opampctl get rolebinding
opampctl create role -f ./role.yaml
opampctl create rolebinding -f ./rolebinding.yaml
```

`create user` flags include `--username`, `--email`, `--password`,
`--password-stdin`, `-f/--file`, and `-o/--output`.

## Version

```bash
opampctl version
```

## Troubleshooting

**Config file not found** — run `opampctl config init` to create a default file, or
pass `--config` with the correct path.

**Authentication errors** — verify your identity with `opampctl whoami`, check the
endpoint with `opampctl config view`, and confirm the user's `auth` block in the
config file. For GitHub users, try `--auth-flow device` if a browser is unavailable.

**Connection issues** — ensure the apiserver is running and reachable at the cluster's
`endpoint`, and that the value matches the running server's `--address`.
