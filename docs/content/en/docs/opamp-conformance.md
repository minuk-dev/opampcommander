---
title: "OpAMP Conformance"
linkTitle: "OpAMP Conformance"
weight: 90
type: docs
---

This page tracks how much of the [OpAMP specification](https://opentelemetry.io/docs/specs/opamp/)
OpAMP Commander implements today. It is a living document; update it whenever the OpAMP message
handling changes.

**Legend:** ✅ Supported · 🟡 Partial · ⛔ Intentionally not advertised / not implemented.
Supported means the described server path exists, not that every collector implements it
or that every normative requirement has an interoperability test. Planned work is not
counted as support.

Capability inventory: all seven nonzero server flags and sixteen nonzero agent flags in
[the OpAMP specification](https://opentelemetry.io/docs/specs/opamp/#agenttoservercapabilities)
and the repository's pinned **opamp-go v0.25.0**. The `Unspecified` zero values mean no
capabilities, not an extra feature. Implementation reviewed at `6905470c5`.

Code references used below:

- [Builder][builder]: `ServerToAgentBuilder.Build`, package offers and connection converters.
- [Handler][handler]: `Service.report`, `OnMessage`, liveness and custom-message dispatch.
- [Agent model][agent-model]: `Report*`, capability checks and desired-state helpers.
- [Converters][converters]: agent report protobuf-to-domain conversions.
- [Group reconciliation][groups]: connection-settings resolution and assignment.

## Server capabilities

The shared [Builder][builder] advertises **six of seven** flags in
`ServerToAgent.capabilities`; `AcceptsConnectionSettingsRequest` is deliberately absent.
These flags describe the server, independently of the capabilities an agent reports.

| ServerCapability | Advertised | Status | Current behavior and code pointer |
|---|:---:|:---:|---|
| `AcceptsStatus` | ✅ | ✅ | Reports are ingested by [Handler][handler] `report()`. |
| `OffersRemoteConfig` | ✅ | ✅ | [Builder][builder] sends the assigned config/hash when [Agent model][agent-model] `HasRemoteConfig` permits it. |
| `AcceptsEffectiveConfig` | ✅ | ✅ | [Handler][handler] stores reports through `ReportEffectiveConfig`. |
| `OffersConnectionSettings` | ✅ | 🟡 | [Builder][builder] emits OpAMP, telemetry and other destinations with headers/TLS; per-agent capability filtering and heartbeat interval negotiation remain incomplete. |
| `AcceptsConnectionSettingsRequest` | ⛔ | ⛔ | [Builder][builder] deliberately withholds the bit because [Handler][handler] does not process requests; no automatic credential issuance. |
| `OffersPackages` | ✅ | 🟡 | [Builder][builder] resolves package offers, but no public assignment path populates the agent's package list; see [Known gaps](#known-gaps). |
| `AcceptsPackagesStatus` | ✅ | ✅ | [Handler][handler] stores package status through `ReportPackageStatuses`. |

## Agent capabilities

Agents advertise these flags; Commander consumes them and the associated reports.
The status column describes **Commander's handling**, not a promise about a particular
collector distribution. Self-telemetry travels to the offered OTLP destinations, not
through the OpAMP endpoint.

| AgentCapability | Status | Current behavior and code pointer |
|---|:---:|---|
| `ReportsStatus` | ✅ | [Handler][handler] records description, capabilities and sequence number through `report()`. |
| `AcceptsRemoteConfig` | ✅ | [Agent model][agent-model] `HasRemoteConfig` gates delivery on this bit. |
| `ReportsEffectiveConfig` | ✅ | [Handler][handler] and [Converters][converters] retain effective configuration reports. |
| `AcceptsPackages` | 🟡 | [Agent model][agent-model] `HasPackagesAvailable` gates offers, but there is no public package assignment workflow. |
| `ReportsPackageStatuses` | ✅ | [Agent model][agent-model] `ReportPackageStatuses` retains the reported package inventory/status. |
| `ReportsOwnTraces` | 🟡 | [Builder][builder] converts `own_traces` destinations, headers and TLS; offers are not filtered by this bit. |
| `ReportsOwnMetrics` | 🟡 | [Builder][builder] converts `own_metrics` destinations, headers and TLS; offers are not filtered by this bit. |
| `ReportsOwnLogs` | 🟡 | [Builder][builder] converts `own_logs` destinations, headers and TLS; offers are not filtered by this bit. |
| `AcceptsOpAMPConnectionSettings` | 🟡 | [Group reconciliation][groups] and [Builder][builder] offer OpAMP endpoints/certificates; the builder does not filter by the current capability bit. |
| `AcceptsOtherConnectionSettings` | 🟡 | [Builder][builder] emits `other_connections`; the builder does not filter by this capability bit. |
| `AcceptsRestartCommand` | 🟡 | [Agent model][agent-model] checks the bit when a restart is requested; [Builder][builder] does not recheck it if the agent later withdraws support, and can combine the command with other fields. |
| `ReportsHealth` | ✅ | [Handler][handler] stores component health including nested subcomponents via [Converters][converters]. |
| `ReportsRemoteConfig` | ✅ | [Agent model][agent-model] `ReportRemoteConfigStatus` records apply status/hash/error; a saved desired config alone is not an acknowledgement. |
| `ReportsHeartbeat` | 🟡 | [Handler][handler] records liveness from heartbeat-only reports; [Builder][builder] does not set `heartbeat_interval_seconds`. |
| `ReportsAvailableComponents` | 🟡 | [Converters][converters] stores component trees and hashes; a hash-only report does not trigger `ReportAvailableComponents` recovery. |
| `ReportsConnectionSettingsStatus` | ✅ | [Handler][handler] stores connection-settings hash/status through `ReportConnectionSettingsStatus`. |

Custom capability strings are separate from these bitmasks. The registry/reply seam is
**partial**: no handlers ship by default and there is no server-originated send API.
See [Custom messages](#custom-messages).

[builder]: https://github.com/minuk-dev/opampcommander/blob/main/pkg/apiserver/domain/agent/service/servertoagent.go
[handler]: https://github.com/minuk-dev/opampcommander/blob/main/pkg/apiserver/application/service/opamp/opamp.go
[agent-model]: https://github.com/minuk-dev/opampcommander/blob/main/pkg/apiserver/domain/agent/agent.go
[converters]: https://github.com/minuk-dev/opampcommander/blob/main/pkg/apiserver/application/service/opamp/protobufsToDomain.go
[groups]: https://github.com/minuk-dev/opampcommander/blob/main/pkg/apiserver/domain/agent/service/agentgroup.go

## Server → Agent message fields

`ServerToAgent` fields the server can populate (`fetchServerToAgent` in `serverToAgent.go`).

| Field | Status | Notes |
|---|:---:|---|
| `instance_uid` | ✅ | |
| `remote_config` | ✅ | Built from the agent's materialized `Spec.RemoteConfig.ConfigMap` with a content hash. |
| `connection_settings` | 🟡 | Destination offers include TLS and headers; capability filtering and heartbeat interval negotiation are incomplete. |
| `packages_available` | ✅ | Package type derived from the spec; an unresolvable package is withheld and logged rather than silently dropped (#496). |
| `agent_identification` | ✅ | Sent when the server assigns a new instance UID. |
| `command` | 🟡 | `Restart` delivery exists, but the builder can include other desired-state fields and does not recheck withdrawn restart support. |
| `capabilities` | ✅ | |
| `flags` (`ReportFullState`) | ✅ | Requested only while the agent's reported info is incomplete (its description or capabilities are still missing); not once it is complete. |
| `error_response` | ✅ | Sent when the server cannot process an `AgentToServer`: `Unavailable` if the agent state cannot be loaded, `BadRequest` if the reported fields cannot be absorbed. Error-only message (no desired-state fields). |
| `custom_capabilities` | 🟡 | Advertises the custom capabilities of the registered custom-message handlers; unset when none are registered (the default). See [Custom messages](#custom-messages). |
| `custom_message` | 🟡 | Sent as a handler's reply to an inbound `custom_message`, gated on the agent having advertised the capability. Never server-originated (no out-of-band send API yet). See [Custom messages](#custom-messages). |

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
| `effective_config` | ✅ | All named entries are stored. For recognized Collector distributions, the `raw` source entry is excluded from active endpoint discovery. |
| `remote_config_status` | ✅ | Stored; `LastUpdatedAt` is stamped from the injected clock. |
| `connection_settings_status` | ✅ | Stored. |
| `package_statuses` | ✅ | Stored. |
| `available_components` | 🟡 | Stores nested sub-components and hashes; hash-only recovery is not implemented. |
| `custom_capabilities` | ✅ | Stored (the agent's declared custom capabilities); used to gate outbound custom messages — see [Custom messages](#custom-messages). |
| `agent_disconnect` | 🟡 | Detected; disconnect is handled via connection-close, not an explicit report. |
| `connection_settings_request` | ⛔ | Not processed; the server withholds `AcceptsConnectionSettingsRequest` rather than advertising it. |
| `custom_message` | 🟡 | Routed to the custom-message handler registered for its capability, if any; dropped when no handler is registered. See [Custom messages](#custom-messages). |

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
3. ~~**Packages: `TopLevel`-only + silent-drop** on fetch failure.~~ *(Resolved in #496.)* The
   advertised `PackageType` now derives from the package spec (`TopLevel`/`AddOn`), and a package
   that cannot be resolved is withheld from the offer and logged (with the agent identity and the
   affected package names) instead of being silently dropped.
4. **Custom messages / custom capabilities** are supported through a [dispatch seam](#custom-messages)
   (#551): inbound `custom_message`s are routed to a handler registered per capability, and a
   handler may reply on the same response. No handlers ship by default, and there is no
   out-of-band (server-originated) send API yet.
5. ~~**`error_response` is never sent.**~~ *(Resolved in #531.)* When the server cannot process an
   incoming `AgentToServer` it now replies with an error-only `ServerToAgent`: `Unavailable`
   when the agent's state cannot be loaded, `BadRequest` when the reported fields cannot be
   absorbed. Processing short-circuits so no partially-applied state is persisted.
6. ~~**Non-string attribute values are dropped** in `toMap`.~~ *(Resolved in #504.)* Non-string
   `AnyValue`s are now preserved in their string form for identifying / non-identifying
   attributes and component metadata.
7. **No way to offer packages to an agent.** `packages_available` is built from the agent's
   `spec.packagesAvailable`, but nothing sets it: the agent update API ignores the field and
   the MongoDB store does not persist it. The capability is advertised, the offer is never sent.

8. **Connection capability negotiation is incomplete.** Group reconciliation assigns a
   connection bundle and the builder converts it without checking each current agent
   capability. The unused `IsOwnMetricsSupported`/`IsOwnTracesSupported` helpers also
   check each other's bits, and `IsOtherConnectionSettingsSupported` checks the OpAMP
   bit. These helpers do not make the active offer path capability-aware.
9. **Heartbeat interval and component hash recovery.** Heartbeats update liveness, but
   the builder does not negotiate `heartbeat_interval_seconds`. Component reports are
   stored, but a hash-only report can replace the stored component map without requesting
   a full `ReportAvailableComponents` response.
10. **Restart message shape and capability changes.** The request path checks restart
    support, but the builder can combine the command with other fields and does not recheck
    withdrawn support. The spec requires a command-only message apart from identity/capabilities.

## Custom messages

OpAMP lets an Agent and Server exchange vendor-specific data outside the standard schema via
`custom_capabilities` and `custom_message` (in both `AgentToServer` and `ServerToAgent`). Both
sides must first advertise a matching custom capability before exchanging the corresponding
custom messages.

Because custom messages are inherently vendor-specific, OpAMP Commander does not hard-code any
custom protocol. Instead it exposes a **dispatch seam** so a feature can plug one in without
touching the core OpAMP handler:

- A `CustomMessageHandler` (`application/service/opamp/custommessage.go`) serves a single
  custom-capability string. Handlers are collected into an FX group
  (`AsCustomMessageHandler`) and indexed by capability in a `CustomMessageRegistry`.
- The server advertises the registered handlers' capabilities in
  `ServerToAgent.custom_capabilities` (via the shared `ServerToAgentBuilder`). With **no handlers
  registered — the default — the field stays unset**, so the wire behavior is identical to the
  previous no-support state.
- An inbound `AgentToServer.custom_message` is routed to the handler registered for its capability.
  A handler may return an outbound `custom_message`, which is merged into the same hot-path
  `ServerToAgent` reply. If no handler is registered for the capability, the message is dropped.
- Outbound custom messages are **capability-gated**: the server never sends a `custom_message` for
  a capability the agent has not advertised (`Agent.HasCustomCapability`).

Delivery is **fire-and-forget with an optional inline reply** — there is no request/response
correlation, and no out-of-band (server-originated) send API yet. The agent's declared
`AgentToServer.custom_capabilities` are also stored as part of its reported state. If you need a
concrete custom protocol wired up, or a server-initiated send API, please open an issue describing
the use case.

## Test coverage

The OpAMP conformance suite (`make test-e2e-conformance`,
`test/e2e/apiserver/conformance_e2e_test.go`) drives the server over the real wire protocol
and asserts the round-trips listed below from both the agent and the REST API side.
It does not prove every capability or every limitation in the matrices above:

| Flow | Reference agent (opamp-go) | OTel Collector `opamp` extension |
|---|:---:|:---:|
| Connect + `AgentDescription` ingest | ✅ WS, HTTP | ✅ WS |
| Server capability negotiation | ✅ | — |
| `ComponentHealth` reporting | ✅ WS, HTTP | — |
| Remote config → `effective_config` round-trip | ✅ WS, HTTP | effective config only |
| Connection settings offer (`own_metrics`) | ✅ WS, HTTP | — |
| `Restart` command round-trip | ✅ WS, HTTP | — |
| `package_statuses` reporting | ✅ WS, HTTP | — |
| `packages_available` offer | ⛔ skipped (gap 7) | — |

The reference agent (`pkg/testutil/opampagent.go`) is the upstream opamp-go client run
in-process, so the test can see exactly what the agent received. The Collector runs as a
testcontainer and covers interop with a real agent, which only implements part of the
protocol (it does not accept remote config or commands).

The message builders and protobuf↔domain converters also have direct, Docker-free unit coverage:
the `AgentToServer` converters in
`pkg/apiserver/application/service/opamp/protobufstodomain_converters_test.go` and the
`ServerToAgent` builder (including the connection-settings offer path) in
`pkg/apiserver/domain/agent/service/servertoagent_test.go`.

### Remote configuration acknowledgment and retry

For agents advertising `ReportsRemoteConfig`, the shared delivery builder stops
reoffering a desired hash once it is reported as `APPLYING`, `APPLIED`, or `FAILED`.
A failed hash remains suppressed when a Supervisor reports `APPLIED` for an older,
restored configuration. Failure information and report timestamps survive server
restart and MongoDB persistence. Agents without status reporting retain redelivery.

`Agent.status.remoteConfig` exposes desired/reported hashes (base64), reported
status, `syncStatus` (`Unassigned`, `Unsupported`, `Pending`, `Applying`, `Applied`,
`Failed`, or `RolledBack`), and the last failure. `RolledBack` means a different hash
was reported applied after the desired hash failed; it is not a claim about the
agent's internal rollback mechanism. An acknowledgment of an older hash is never
proof that the current desired configuration was applied.

Change the assigned configuration to deliver a new hash, or explicitly retry the
unchanged configuration with `spec.retryRemoteConfig: true` in the existing agent
update API. This write-only command clears acknowledgment/failure state, persists
it, and notifies the agent. Normal reconciliation and reconnects do not clear it.
An agent reporting `UNSET` after losing local state can receive its desired config
again, unless that desired hash previously failed. Retry is not a revision barrier:
late reports for the same hash may still describe the earlier attempt.
### Collector identity across restarts

OpAMP `instance_uid` and the Collector resource's `service.instance.id` are
independent. A changed description on an existing session, after disconnect, or
on a new HTTP poll does not allocate a new UID. Duplicate-session detection
requires a different live WebSocket plus a conflicting reported identity.
Overlapping WebSocket reconnects with the same identity are accepted. Without
authenticated identity, simultaneous sessions reporting different identities
cannot reliably be distinguished from a restart whose old socket is still open;
clients should close the old socket before reconnecting with a changed identity.
This heuristic does not replace mTLS certificate authorization.
### Namespace attributes

`service.namespace` is read from non-empty identifying attributes first, falling
back to non-identifying attributes for Collector resource reports. Missing or
empty attributes retain the existing namespace (the configured default for a new
agent). Description updates reconcile matching groups and update connection
namespace metadata in the same report cycle; certificate authorization is unchanged.
