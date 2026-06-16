---
title: "API Reference"
linkTitle: "API"
weight: 3
type: docs
description: >
  The OpAMP Commander REST API.
---

The apiserver exposes a REST API under `/api/v1`. Most resources are
**namespace-scoped** and live under `/api/v1/namespaces/{namespace}/...`; a few
(hosts, containers, roles, users, server info) are cluster-scoped.

Interactive API documentation (Swagger UI) is generated from the source and served by
the running server. The OpAMP agent protocol itself is handled over a WebSocket at
`/api/v1/opamp`.

## Authentication

Obtain a JWT and send it as a bearer token:

```http
Authorization: Bearer <your-jwt-token>
```

### Basic auth

```http
GET /api/v1/auth/basic
Authorization: Basic <base64(username:password)>
```

### GitHub OAuth2 (browser)

```http
GET  /api/v1/auth/github                 # begin browser-based login
POST /api/v1/auth/github/authcode        # exchange an authorization code
```

### GitHub device flow (CLI)

```http
GET /api/v1/auth/github/device           # request a device + user code
GET /api/v1/auth/github/device/exchange  # poll to exchange for a token
```

### Session helpers

```http
GET /api/v1/auth/info       # info about the current credential
GET /api/v1/auth/refresh    # refresh an access token
```

## Namespaces

```http
GET    /api/v1/namespaces
POST   /api/v1/namespaces
GET    /api/v1/namespaces/{namespace}
DELETE /api/v1/namespaces/{namespace}
```

A namespace is derived from each agent's `service.namespace` identifying attribute,
defaulting to `default`.

## Agents

```http
GET  /api/v1/namespaces/{namespace}/agents
GET  /api/v1/namespaces/{namespace}/agents/{id}
POST /api/v1/namespaces/{namespace}/agents/search
```

List endpoints accept `limit` and `continue` query parameters for pagination.

## Agent groups

```http
GET    /api/v1/namespaces/{namespace}/agentgroups
POST   /api/v1/namespaces/{namespace}/agentgroups
GET    /api/v1/namespaces/{namespace}/agentgroups/{name}
PUT    /api/v1/namespaces/{namespace}/agentgroups/{name}
DELETE /api/v1/namespaces/{namespace}/agentgroups/{name}
GET    /api/v1/namespaces/{namespace}/agentgroups/{name}/agents
```

## Agent packages

```http
GET    /api/v1/namespaces/{namespace}/agentpackages
POST   /api/v1/namespaces/{namespace}/agentpackages
GET    /api/v1/namespaces/{namespace}/agentpackages/{name}
DELETE /api/v1/namespaces/{namespace}/agentpackages/{name}
```

## Agent remote configs

```http
GET    /api/v1/namespaces/{namespace}/agentremoteconfigs
POST   /api/v1/namespaces/{namespace}/agentremoteconfigs
GET    /api/v1/namespaces/{namespace}/agentremoteconfigs/{name}
DELETE /api/v1/namespaces/{namespace}/agentremoteconfigs/{name}
```

## Certificates

```http
GET    /api/v1/namespaces/{namespace}/certificates
POST   /api/v1/namespaces/{namespace}/certificates
GET    /api/v1/namespaces/{namespace}/certificates/{name}
DELETE /api/v1/namespaces/{namespace}/certificates/{name}
```

## Connections

```http
GET /api/v1/namespaces/{namespace}/connections
```

Returns the active agent connections for a namespace.

## Hosts and containers (cluster-scoped)

```http
GET /api/v1/hosts
GET /api/v1/hosts/{id}
GET /api/v1/hosts/{id}/agents

GET /api/v1/containers
GET /api/v1/containers/{id}
GET /api/v1/containers/{id}/agents
```

## RBAC

```http
GET    /api/v1/roles
POST   /api/v1/roles
GET    /api/v1/roles/{id}

GET    /api/v1/namespaces/{namespace}/rolebindings
POST   /api/v1/namespaces/{namespace}/rolebindings
GET    /api/v1/namespaces/{namespace}/rolebindings/{name}
DELETE /api/v1/namespaces/{namespace}/rolebindings/{name}
```

## Users

```http
GET  /api/v1/users
POST /api/v1/users
GET  /api/v1/users/me
GET  /api/v1/users/{id}
```

## Server information

```http
GET /api/v1/servers     # cluster server info
GET /api/v1/version      # build version
GET /api/v1/ping         # connectivity check
```

## Health checks

Served by the management server (default `localhost:9090`):

```http
GET /healthz
GET /readyz
```

## Error responses

Errors follow the [RFC 9457 Problem Details](https://www.rfc-editor.org/rfc/rfc9457)
format:

```json
{
  "type": "about:blank",
  "title": "Not Found",
  "status": 404,
  "detail": "resource does not exist"
}
```

**Common status codes:**

- `200 OK` — request succeeded
- `201 Created` — resource created
- `204 No Content` — succeeded with no body
- `400 Bad Request` — invalid parameters
- `401 Unauthorized` — missing or invalid authentication
- `404 Not Found` — resource not found
- `500 Internal Server Error` — server error
