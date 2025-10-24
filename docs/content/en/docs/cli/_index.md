---
title: "CLI Reference"
linkTitle: "CLI"
weight: 4
type: docs
description: >
  Learn how to use the opampctl command line tool.
---

`opampctl` is a command-line interface for interacting with the OpAMP Commander server. It provides commands for managing agents, agent groups, and configurations.

## Installation

Install `opampctl` using Go:

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

Before using `opampctl`, you need to initialize the configuration:

```bash
opampctl config init
```

This creates a configuration file at `~/.config/opampcommander/opampctl/config.yaml`.

### Configuration File

The configuration file stores server connection details and authentication information:

```yaml
contexts:
  - name: production
    server: https://opamp.example.com
    token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
  - name: staging
    server: https://opamp-staging.example.com
    token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...

currentContext: production
```

### Configuration Commands

#### Initialize Configuration

```bash
opampctl config init
```

Creates a default configuration file.

#### View Configuration

```bash
opampctl config view
```

Displays the current configuration.

## Context Management

Contexts allow you to switch between different OpAMP Commander servers.

### List Contexts

```bash
opampctl context ls
```

Shows all available contexts and highlights the current one.

### Use Context

```bash
opampctl context use <context-name>
```

Switch to a different context.

**Example:**
```bash
opampctl context use staging
```

## Authentication

### Check Authentication Status

```bash
opampctl whoami
```

Displays information about the currently authenticated user or service account.

## Agent Management

### List Agents

```bash
opampctl get agent
opampctl get agents
```

Lists all agents connected to the OpAMP Commander server.

**Options:**
- `--limit <n>`: Limit the number of results
- `--continue <token>`: Continue from a previous pagination token

**Example:**
```bash
opampctl get agent --limit 20
```

**Output:**
```
INSTANCE UID              SERVICE NAME    STATUS    LAST SEEN
agent-001                 my-service      active    2024-01-01T00:00:00Z
agent-002                 other-service   active    2024-01-01T00:05:00Z
```

### Get Agent Details

```bash
opampctl get agent <instance-uid>
```

Displays detailed information about a specific agent.

**Example:**
```bash
opampctl get agent agent-001
```

**Output:**
```yaml
instanceUid: agent-001
description:
  identifyingAttributes:
    service.name: my-service
    service.instance.id: instance-1
  nonIdentifyingAttributes:
    host.name: server-1
capabilities: 15
isManaged: true
effectiveConfig:
  configMap:
    collector.yaml:
      body: |
        receivers:
          otlp:
            protocols:
              grpc:
      contentType: text/yaml
```

## Agent Group Management

### List Agent Groups

```bash
opampctl get agentgroup
opampctl get agentgroups
```

Lists all agent groups.

**Example:**
```bash
opampctl get agentgroup
```

**Output:**
```
NAME          DESCRIPTION              CREATED
production    Production agents        2024-01-01T00:00:00Z
staging       Staging agents           2024-01-01T00:05:00Z
```

### Get Agent Group Details

```bash
opampctl get agentgroup <name>
```

Displays detailed information about a specific agent group.

### Create Agent Group

```bash
opampctl create agentgroup <name> [flags]
```

Creates a new agent group.

**Flags:**
- `--description <text>`: Description of the agent group
- `--label <key=value>`: Add labels (can be specified multiple times)
- `--config-file <path>`: Path to configuration file

**Example:**
```bash
opampctl create agentgroup production \
  --description "Production environment agents" \
  --label env=production \
  --label tier=critical \
  --config-file ./collector-config.yaml
```

### Delete Agent Group

```bash
opampctl delete agentgroup <name>
```

Deletes an agent group.

**Example:**
```bash
opampctl delete agentgroup staging
```

## Connection Management

### List Active Connections

```bash
opampctl get connection
opampctl get connections
```

Lists all active agent connections to the server.

**Output:**
```
INSTANCE UID    REMOTE ADDRESS         CONNECTED AT              LAST HEARTBEAT
agent-001       192.168.1.100:54321    2024-01-01T00:00:00Z     2024-01-01T00:05:00Z
agent-002       192.168.1.101:54322    2024-01-01T00:01:00Z     2024-01-01T00:06:00Z
```

## Global Flags

The following flags are available for all commands:

- `--config <path>`: Path to configuration file (default: `~/.config/opampcommander/opampctl/config.yaml`)
- `--context <name>`: Use a specific context
- `--server <url>`: OpAMP Commander server URL (overrides context)
- `--token <token>`: Authentication token (overrides context)
- `--output <format>`: Output format (json, yaml, table) - default: table
- `--verbose`: Enable verbose output
- `--help`: Display help information

## Version

```bash
opampctl version
```

Displays the version information for `opampctl`.

**Output:**
```
opampctl version: v1.0.0
commit: abc123def456
build date: 2024-01-01T00:00:00Z
```

## Examples

### Complete Workflow Example

1. Initialize configuration:
```bash
opampctl config init
```

2. Check authentication:
```bash
opampctl whoami
```

3. List all agents:
```bash
opampctl get agents
```

4. Create a new agent group:
```bash
opampctl create agentgroup dev \
  --description "Development environment" \
  --label env=dev
```

5. View agent group details:
```bash
opampctl get agentgroup dev
```

6. List active connections:
```bash
opampctl get connections
```

### Working with Multiple Environments

```bash
# Switch to staging environment
opampctl context use staging

# List agents in staging
opampctl get agents

# Switch back to production
opampctl context use production

# List agents in production
opampctl get agents
```

### Using Different Output Formats

```bash
# JSON output
opampctl get agent agent-001 --output json

# YAML output
opampctl get agent agent-001 --output yaml

# Table output (default)
opampctl get agent agent-001
```

## Troubleshooting

### Configuration File Not Found

If you see an error about the configuration file not found:

```bash
opampctl config init
```

### Authentication Errors

If you encounter authentication errors:

1. Check your token is valid:
```bash
opampctl whoami
```

2. Verify your server URL is correct:
```bash
opampctl config view
```

3. Re-authenticate if necessary using the API authentication endpoints.

### Connection Issues

If you can't connect to the server:

1. Verify the server is running and accessible
2. Check your network connectivity
3. Ensure the server URL in your configuration is correct
4. Try using the `--server` flag to override the configuration

## Exit Codes

- `0`: Success
- `1`: General error
- `2`: Configuration error
- `3`: Authentication error
- `4`: Connection error
- `5`: Resource not found
