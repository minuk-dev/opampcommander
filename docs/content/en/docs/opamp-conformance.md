---
title: "OpAMP Conformance"
linkTitle: "OpAMP Conformance"
weight: 90
type: docs
---

This page tracks how much of the [OpAMP specification](https://opentelemetry.io/docs/specs/opamp/)
OpAMP Commander implements today. It is a living document; update it whenever the OpAMP message
handling changes.

**Legend:** ✅ Implemented · 🟡 Partial · ⛔ Not implemented

> Audited against `main` on 2026-07-21. Primary sources:
> `pkg/apiserver/application/service/opamp/{opamp,serverToAgent,protobufsToDomain,domainToProtobufs}.go`
> and `pkg/apiserver/domain/agent/service/server.go`.

## Server capabilities

The value advertised in `ServerToAgent.capabilities` (`serverToAgent.go`). All seven server
capabilities are advertised; "Fulfilled" reflects whether the behavior behind the flag is
actually implemented.

| ServerCapability | Advertised | Fulfilled | Notes |
|---|:---:|:---:|---|
| `AcceptsStatus` | ✅ | ✅ | Full `AgentToServer` ingestion via `report()`. |
| `OffersRemoteConfig` | ✅ | ✅ | Delivered by the shared `ServerToAgentBuilder` on both the hot path and the cross-server push path. |
| `AcceptsEffectiveConfig` | ✅ | ✅ | Stored on the agent. |
| `OffersConnectionSettings` | ✅ | ✅ | `opamp`, `own_metrics`, `own_logs`, `own_traces`, `other_connections`, each with headers + TLS certificate. |
| `AcceptsConnectionSettingsRequest` | ⛔ | ⛔ | Intentionally **not advertised**: `connection_settings_request` is not processed, so the capability is withheld rather than claimed-but-ignored. |
| `OffersPackages` | ✅ | 🟡 | Only `TopLevel` package type; a package-fetch failure is silently dropped (see #496). |
| `AcceptsPackagesStatus` | ✅ | ✅ | Stored on the agent. |

## Server → Agent message fields

`ServerToAgent` fields the server can populate (`fetchServerToAgent` in `serverToAgent.go`).

| Field | Status | Notes |
|---|:---:|---|
| `instance_uid` | ✅ | |
| `remote_config` | ✅ | Built from the agent's materialized `Spec.RemoteConfig.ConfigMap` with a content hash. |
| `connection_settings` | ✅ | Full offer set incl. TLS certificates and headers. |
| `packages_available` | 🟡 | `TopLevel` only; silent drop on fetch failure (#496). |
| `agent_identification` | ✅ | Sent when the server assigns a new instance UID. |
| `command` | ✅ | `Restart` — the only command the spec currently defines. |
| `capabilities` | ✅ | |
| `flags` (`ReportFullState`) | ✅ | Requested only while the agent's reported info is incomplete (its description or capabilities are still missing); not once it is complete. |
| `error_response` | ✅ | Sent when the server cannot process an `AgentToServer`: `Unavailable` if the agent state cannot be loaded, `BadRequest` if the reported fields cannot be absorbed. Error-only message (no desired-state fields). |
| `custom_capabilities` | ⛔ | Always `nil` — [intentionally unsupported](#custom-messages). |
| `custom_message` | ⛔ | Always `nil` — [intentionally unsupported](#custom-messages). |

## Agent → Server processing

`AgentToServer` fields the server ingests (`report()` in `opamp.go`, converters in
`protobufsToDomain.go`).

| Field | Status | Notes |
|---|:---:|---|
| `instance_uid` | ✅ | Incl. instance-UID conflict handling (`instanceuid_conflict.go`). |
| `sequence_num` | ✅ | Recorded per report. |
| `agent_description` | ✅ | Ingested; non-string `AnyValue`s are preserved in their string form (`protobufsToDomain.go`, `anyValueToString`). |
| `capabilities` | ✅ | Stored. |
| `health` (`ComponentHealth`) | ✅ | Incl. nested sub-component health. |
| `effective_config` | ✅ | Stored. |
| `remote_config_status` | ✅ | Stored; `LastUpdatedAt` is stamped from the injected clock. |
| `connection_settings_status` | ✅ | Stored. |
| `package_statuses` | ✅ | Stored. |
| `available_components` | ✅ | Incl. nested sub-components. |
| `custom_capabilities` | ✅ | Stored (the agent's declared custom capabilities), but not acted on — see [Custom messages](#custom-messages). |
| `agent_disconnect` | 🟡 | Detected; disconnect is handled via connection-close, not an explicit report. |
| `connection_settings_request` | ⛔ | Not processed; the server withholds `AcceptsConnectionSettingsRequest` rather than advertising it. |
| `custom_message` | ⛔ | [Intentionally not processed](#custom-messages) — dropped. |

## Known gaps

1. ~~**Two divergent `ServerToAgent` builders.**~~ *(Resolved in #503.)* The hot path and the
   cross-server push path now share a single `ServerToAgentBuilder`, so a cross-server push
   delivers the same complete message (remote config, packages, connection settings, command)
   as a direct response instead of a degraded stub. Previously the push path
   (`buildServerToAgentMessage`) omitted those fields and, for a fully-described agent, left
   `ReportFullState` unset — making the immediate push effectively a no-op until the agent's
   next heartbeat.
2. ~~**`AcceptsConnectionSettingsRequest` advertised but unhandled.**~~ *(Resolved.)* The server
   no longer advertises the capability, so agents are not invited to send a
   `connection_settings_request` the server would silently ignore. Re-advertise once the request
   is processed.
3. **Packages: `TopLevel`-only + silent-drop** on fetch failure (tracked in #496).
4. **Custom messages / custom capabilities** are [intentionally unsupported](#custom-messages)
   (documented decision, not an oversight).
5. ~~**`error_response` is never sent.**~~ *(Resolved in #531.)* When the server cannot process an
   incoming `AgentToServer` it now replies with an error-only `ServerToAgent`: `Unavailable`
   when the agent's state cannot be loaded, `BadRequest` when the reported fields cannot be
   absorbed. Processing short-circuits so no partially-applied state is persisted.
6. ~~**Non-string attribute values are dropped** in `toMap`.~~ *(Resolved in #504.)* Non-string
   `AnyValue`s are now preserved in their string form for identifying / non-identifying
   attributes and component metadata.

## Custom messages

OpAMP lets an Agent and Server exchange vendor-specific data outside the standard schema via
`custom_capabilities` and `custom_message` (in both `AgentToServer` and `ServerToAgent`). Both
sides must first advertise a matching custom capability before exchanging the corresponding
custom messages.

**OpAMP Commander intentionally does not support custom message exchange.** This is a deliberate
decision, not an unimplemented gap:

- The server advertises **no** custom capabilities, so it never populates
  `ServerToAgent.custom_capabilities` and never originates a `ServerToAgent.custom_message`.
- An incoming `AgentToServer.custom_message` is **dropped** (not stored or routed).
- The agent's declared `AgentToServer.custom_capabilities` *are* stored as part of the agent's
  reported state, but the server takes no action on them.

Custom messages are inherently vendor-specific: supporting them would mean defining and
maintaining server-side semantics for message types the OpAMP spec deliberately leaves open.
OpAMP Commander targets **general-purpose, spec-standard** agent management, so this is out of
scope until a concrete, broadly-useful custom protocol justifies it. If you need custom message
exchange, please open an issue describing the use case.

## Test coverage

Most OpAMP message-path behavior is exercised only by the Docker-based E2E suite
(`test/e2e/apiserver`). The message builders and protobuf↔domain converters have little direct
unit coverage. Interop against a real OTel Collector `opamp` extension is a goal (a Collector
test helper already exists in `pkg/testutil`).
