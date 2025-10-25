---
title: "API Reference"
linkTitle: "API"
weight: 3
type: docs
description: >
  Learn how to use the OpAMP Commander REST API.
---

OpAMP Commander provides a RESTful API for managing agents, agent groups, and configurations.

## Authentication

Most API endpoints require authentication. OpAMP Commander supports multiple authentication methods:

### Basic Authentication

```http
GET /api/v1/auth/basic
Authorization: Basic <base64-encoded-credentials>
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expiresAt": "2024-01-01T00:00:00Z"
}
```

### GitHub OAuth2

```http
GET /api/v1/auth/github
```

Returns the GitHub OAuth2 authorization URL for web-based authentication.

### GitHub Device Flow

For CLI tools, use the device authorization flow:

```http
GET /api/v1/auth/github/device
```

**Response:**
```json
{
  "device_code": "...",
  "user_code": "ABCD-1234",
  "verification_uri": "https://github.com/login/device",
  "expires_in": 900,
  "interval": 5
}
```

Exchange the device code for a token:

```http
GET /api/v1/auth/github/device/exchange?device_code=<device-code>
```

### Using Authentication Token

Include the token in the Authorization header for subsequent requests:

```http
Authorization: Bearer <your-jwt-token>
```

## Agent Management

### List Agents

Retrieve a list of all connected agents.

```http
GET /api/v1/agents?limit=10&continue=<token>
```

**Query Parameters:**
- `limit` (optional): Maximum number of agents to return
- `continue` (optional): Continuation token for pagination

**Response:**
```json
[
  {
    "instanceUid": "agent-001",
    "description": {
      "identifyingAttributes": {
        "service.name": "my-service",
        "service.instance.id": "instance-1"
      },
      "nonIdentifyingAttributes": {
        "host.name": "server-1"
      }
    },
    "capabilities": 15,
    "isManaged": true,
    "effectiveConfig": {
      "configMap": {
        "config.yaml": {
          "body": "...",
          "contentType": "text/yaml"
        }
      }
    },
    "remoteConfig": {...},
    "componentHealth": {...},
    "packageStatuses": {...}
  }
]
```

### Get Agent

Retrieve details for a specific agent by instance UID.

```http
GET /api/v1/agents/{instanceUid}
```

**Response:** Returns a single Agent object.

### Update Agent Configuration

Send a configuration update command to an agent.

```http
POST /api/v1/agents/{instanceUid}/update-agent-config
Content-Type: application/json

{
  "remoteConfig": {
    "config": {
      "configMap": {
        "collector.yaml": {
          "body": "receivers:\n  otlp:\n    protocols:\n      grpc:\n",
          "contentType": "text/yaml"
        }
      }
    },
    "configHash": "..."
  }
}
```

**Response:**
```json
{
  "id": "cmd-123",
  "kind": "update-config",
  "targetInstanceUid": "agent-001",
  "data": {...}
}
```

## Agent Groups

Agent groups allow you to manage configurations for multiple agents collectively.

### List Agent Groups

```http
GET /api/v1/agentgroups?limit=10&continue=<token>
```

**Query Parameters:**
- `limit` (optional): Maximum number of groups to return
- `continue` (optional): Continuation token for pagination

**Response:**
```json
[
  {
    "name": "production",
    "description": "Production environment agents",
    "labels": {
      "env": "production"
    },
    "remoteConfig": {...},
    "createdAt": "2024-01-01T00:00:00Z",
    "updatedAt": "2024-01-01T00:00:00Z"
  }
]
```

### Get Agent Group

```http
GET /api/v1/agentgroups/{name}
```

**Response:** Returns a single AgentGroup object.

### Create Agent Group

```http
POST /api/v1/agentgroups
Content-Type: application/json

{
  "name": "staging",
  "description": "Staging environment agents",
  "labels": {
    "env": "staging"
  },
  "remoteConfig": {
    "config": {
      "configMap": {
        "collector.yaml": {
          "body": "...",
          "contentType": "text/yaml"
        }
      }
    }
  }
}
```

**Response:** Returns the created AgentGroup with status code 201.

### Update Agent Group

```http
PUT /api/v1/agentgroups/{name}
Content-Type: application/json

{
  "name": "staging",
  "description": "Updated description",
  "labels": {
    "env": "staging",
    "version": "v2"
  },
  "remoteConfig": {...}
}
```

**Response:** Returns the updated AgentGroup.

### Delete Agent Group

```http
DELETE /api/v1/agentgroups/{name}
```

**Response:** 204 No Content on success.

## Commands

### List Commands

View all commands sent to agents.

```http
GET /api/v1/commands?limit=10&continue=<token>
```

**Response:**
```json
[
  {
    "id": "cmd-123",
    "kind": "update-config",
    "targetInstanceUid": "agent-001",
    "data": {...},
    "createdAt": "2024-01-01T00:00:00Z",
    "status": "pending"
  }
]
```

### Get Command

```http
GET /api/v1/commands/{id}
```

**Response:** Returns a single AgentCommand object.

## Connections

### List Active Connections

View all active agent connections.

```http
GET /api/v1/connections
```

**Response:**
```json
[
  {
    "instanceUid": "agent-001",
    "remoteAddr": "192.168.1.100:54321",
    "connectedAt": "2024-01-01T00:00:00Z",
    "lastHeartbeat": "2024-01-01T00:05:00Z"
  }
]
```

## Server Information

### Get Server Info

```http
GET /api/v1/servers
```

Returns information about the OpAMP Commander server.

### Version

```http
GET /api/v1/version
```

**Response:**
```json
{
  "version": "v1.0.0",
  "commit": "abc123",
  "buildDate": "2024-01-01T00:00:00Z"
}
```

### Health Checks

```http
GET /healthz
GET /readyz
```

Returns 200 OK if the server is healthy/ready.

### Ping

```http
GET /api/v1/ping
```

Returns 200 OK for connectivity testing.

## Error Responses

All endpoints return standard HTTP status codes. Error responses follow this format:

```json
{
  "error": "Error message description",
  "code": "ERROR_CODE",
  "details": {...}
}
```

**Common Status Codes:**
- `200 OK`: Request succeeded
- `201 Created`: Resource created successfully
- `204 No Content`: Request succeeded with no response body
- `400 Bad Request`: Invalid request parameters
- `401 Unauthorized`: Authentication required or invalid
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server error occurred
