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

> Audited against `main` on 2026-07-19. Primary sources:
> `pkg/apiserver/application/service/opamp/{opamp,serverToAgent,protobufsToDomain,domainToProtobufs}.go`
> and `pkg/apiserver/domain/agent/service/server.go`.

## Server capabilities

The value advertised in `ServerToAgent.capabilities` (`serverToAgent.go`). All seven server
capabilities are advertised; "Fulfilled" reflects whether the behavior behind the flag is
actually implemented.

| ServerCapability | Advertised | Fulfilled | Notes |
|---|:---:|:---:|---|
| `AcceptsStatus` | ✅ | ✅ | Full `AgentToServer` ingestion via `report()`. |
| `OffersRemoteConfig` | ✅ | ✅ | Delivered on the hot path; **omitted** on the cross-server push path (see [Known gaps](#known-gaps)). |
| `AcceptsEffectiveConfig` | ✅ | ✅ | Stored on the agent. |
| `OffersConnectionSettings` | ✅ | ✅ | `opamp`, `own_metrics`, `own_logs`, `own_traces`, `other_connections`, each with headers + TLS certificate. |
| `AcceptsConnectionSettingsRequest` | ✅ | ⛔ | Advertised, but `connection_settings_request` from the agent is **not processed**. |
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
| `flags` (`ReportFullState`) | ⛔ | Intentionally never set. Soliciting it on every hot-path response drives an agent re-report loop (`NeedFullStateCommand` is true for any agent without a pending instance-UID change), so the server does not request full state. |
| `error_response` | ⛔ | Never populated. |
| `custom_capabilities` | ⛔ | Always `nil` (server advertises no custom capabilities). |
| `custom_message` | ⛔ | Always `nil`. |

## Agent → Server processing

`AgentToServer` fields the server ingests (`report()` in `opamp.go`, converters in
`protobufsToDomain.go`).

| Field | Status | Notes |
|---|:---:|---|
| `instance_uid` | ✅ | Incl. instance-UID conflict handling (`instanceuid_conflict.go`). |
| `sequence_num` | ✅ | Recorded per report. |
| `agent_description` | ✅ | Ingested; every `AnyValue` kind is preserved as its string form (#504). Attributes are stored as `map[string]string`, so structured array/kvlist values keep a best-effort textual form. |
| `capabilities` | ✅ | Stored. |
| `health` (`ComponentHealth`) | ✅ | Incl. nested sub-component health. |
| `effective_config` | ✅ | Stored. |
| `remote_config_status` | ✅ | Stored (uses `time.Now()` rather than the injected clock — hygiene nit). |
| `connection_settings_status` | ✅ | Stored. |
| `package_statuses` | ✅ | Stored. |
| `available_components` | ✅ | Incl. nested sub-components. |
| `custom_capabilities` | ✅ | Stored. |
| `agent_disconnect` | 🟡 | Detected; disconnect is handled via connection-close, not an explicit report. |
| `connection_settings_request` | ⛔ | Not processed despite `AcceptsConnectionSettingsRequest` being advertised. |
| `custom_message` | ⛔ | Not processed. |

## Known gaps

1. ~~**Two divergent `ServerToAgent` builders.**~~ *(Resolved in #503.)* The hot path and the
   cross-server push path now share a single `ServerToAgentBuilder`, so a cross-server push
   delivers the same complete message (remote config, packages, connection settings, command)
   as a direct response instead of a degraded stub. Previously the push path
   (`buildServerToAgentMessage`) omitted those fields and, for a fully-described agent, left
   `ReportFullState` unset — making the immediate push effectively a no-op until the agent's
   next heartbeat.
2. **`AcceptsConnectionSettingsRequest` advertised but unhandled** — the server claims the
   capability but ignores `connection_settings_request`.
3. **Packages: `TopLevel`-only + silent-drop** on fetch failure (tracked in #496).
4. **Custom messages / custom capabilities** are unsupported end-to-end (server→agent always
   `nil`; agent→server `custom_message` dropped).
5. **`error_response` is never sent**, so the agent gets no structured error signal from the server.
6. ~~**Non-string attribute values are dropped** in `toMap`.~~ *(Resolved in #504.)* Every
   `AnyValue` kind is now rendered to its string form, so non-string identifying attributes
   (e.g. an int `process.pid`) are preserved and still participate in AgentGroup selector
   matching.

## Test coverage

Most OpAMP message-path behavior is exercised only by the Docker-based E2E suite
(`test/e2e/apiserver`). The message builders and protobuf↔domain converters have little direct
unit coverage. Interop against a real OTel Collector `opamp` extension is a goal (a Collector
test helper already exists in `pkg/testutil`).
