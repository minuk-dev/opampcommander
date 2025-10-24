---
title: "API Reference"
linkTitle: "API"
weight: 3
type: docs
description: >
  Learn how to use the OpAMP Commander API.
---

## REST API

OpAMP Commander provides a RESTful API.

### List Agents

```http
GET /api/v1/agents
```

**Response Example:**

```json
{
  "agents": [
    {
      "id": "agent-001",
      "name": "collector-1",
      "status": "active",
      "last_seen": "2024-01-01T00:00:00Z"
    }
  ]
}
```

### Get Agent Details

```http
GET /api/v1/agents/{agentId}
```

### Update Agent Configuration

```http
PUT /api/v1/agents/{agentId}/config
Content-Type: application/json

{
  "config": {
    "receivers": {...},
    "processors": {...},
    "exporters": {...}
  }
}
```

## gRPC API

The gRPC API implements the OpAMP protocol. For more details, refer to the [OpAMP Spec](https://github.com/open-telemetry/opamp-spec).

## Authentication

API requests require a JWT token:

```http
Authorization: Bearer <your-jwt-token>
```
