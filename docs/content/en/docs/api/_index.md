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

List endpoints accept `limit` and `continue` query parameters for pagination, plus
the filtering parameters described in [Filtering and searching](#filtering-and-searching).

## Filtering and searching

Every list endpoint accepts three filtering parameters. They are answered by the
datastore, not by the server fetching a page and discarding rows from it, so
`remainingItemCount` always describes the same set the page was drawn from.

```http
GET /api/v1/namespaces/prod/agents?attributeSelector=service.name%3Dotel-collector&fieldSelector=status.connected%3Dtrue
GET /api/v1/namespaces/prod/endpoints?labelSelector=env%3Dprod
```

### `labelSelector` and `attributeSelector`

Two parameters, one grammar. They are separate because they read different
metadata, and the difference is **who sets it**:

- **`labelSelector`** reads metadata *an operator set* through this API and can
  change.
- **`attributeSelector`** reads metadata *the resource reported about itself*. Only
  agents have any: their OpAMP `AgentDescription`, which arrives over the protocol
  and cannot be set here.

Sending the one a resource does not have is a `400` naming what it does have — not
an ignored parameter, which would hand back the whole collection with a `200`.

The expression syntax below is identical for both. Multiple requirements are
comma-separated and combined with AND.

| Form | Example | Matches |
|---|---|---|
| `key=value`, `key==value` | `env=prod` | the label is present and equal |
| `key!=value` | `tier!=canary` | the label differs **or is absent** |
| `key in (a,b)` | `env in (prod,stg)` | the label is one of the listed values |
| `key notin (a,b)` | `tier notin (canary,dev)` | the label is none of them, **or is absent** |
| `key` | `deprecated` | the label is present, whatever its value |
| `!key` | `!deprecated` | the label is absent |

The negative operators deliberately match a resource that carries no such label at
all, exactly as Kubernetes label selectors do.

Which parameter a resource answers, and what backs it:

| Resource | Parameter | Backed by |
|---|---|---|
| namespaces, hosts, containers, users | `labelSelector` | `metadata.labels` |
| agent groups, packages, remote configs, certificates, endpoints, schemas | `labelSelector` | `metadata.attributes` |
| agents | `attributeSelector` | the reported agent description |
| roles, role bindings | neither | — |

An agent's description is one attribute domain. The split into identifying and
non-identifying attributes says which attributes form the agent's *identity*, not
which an operator may *filter* on, so `attributeSelector=os.type%3Dlinux` finds an
agent whichever half reports it.

Roles and role bindings carry neither kind. Either parameter against them is
answered with `400 Bad Request` saying so, rather than with an empty list.

### `fieldSelector`

An expression over the resource's own fields, using `=`, `==` and `!=` only —
there are no set-based or existence forms. Only the fields a resource documents as
selectable may be referenced; anything else is a `400` naming the field and listing
the supported ones, so a filter is never silently dropped.

| Resource | Selectable fields |
|---|---|
| agents | `metadata.namespace`, `status.connected`, `status.healthy` |
| agent groups, remote configs, certificates | `metadata.namespace` |
| agent packages | `metadata.namespace`, `spec.packageType`, `spec.version` |
| remote config schemas | `metadata.namespace`, `spec.binary`, `spec.version` |
| endpoints | `metadata.namespace`, `spec.protocol` |
| hosts, containers | `spec.platform` |
| namespaces | `metadata.name` |
| users | `spec.isActive` |
| roles | `spec.isBuiltIn` |
| role bindings | `spec.roleRef.name` |

`status.connected` uses the same staleness-aware predicate as the connected badge,
so a filtered list and the badge cannot disagree. The `connected=true` parameter on
the agents endpoint is a shorthand for `fieldSelector=status.connected=true`.

### `name` and `nameContains`

```http
GET /api/v1/namespaces/prod/endpoints?name=tempo-
GET /api/v1/namespaces/prod/endpoints?nameContains=tempo
```

`name` is a **case-sensitive prefix**, served by an index range scan. `nameContains`
is a **case-insensitive substring**; no ordered index can answer "contains", so it is
a scan — prefer `name` where a prefix will do. Both are matched literally: neither is
interpreted as a pattern. They combine, which lets you anchor a scan to an indexed
prefix.

For agents, the name is the instance UID; for roles, the display name; for users,
the email.

### Deprecated parameters

The agents endpoint still accepts `selector` and `nonIdentifyingSelector`, repeated
`key=value` parameters that predate `attributeSelector`. They are deprecated:
`attributeSelector` expresses the same equality form and adds `!=`, `in`, `notin`
and existence, and reaches both halves of the agent description.

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
