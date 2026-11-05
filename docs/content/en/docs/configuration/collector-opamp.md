---
title: "Collector and Supervisor connections"
weight: 20
---

Commander serves OpAMP WebSocket and HTTP traffic at `/api/v1/opamp`. Choose the
connection topology according to the capabilities you need. The configurations
below target Collector Contrib **0.163.1** (Contrib components 0.163.0) and
Supervisor **0.163.0**. Collector 0.163.1 fixes the 0.163.0 distribution release.

| Deployment | Behavior |
| --- | --- |
| Collector with the OpAMP extension connected directly | Reports description, health, effective config and available components. Does not accept remote configuration. |
| Supervisor connected to Commander, with a managed Collector | Applies remote configuration when `accepts_remote_config` is enabled; reports its status and can restore a last-working configuration. Supervisor injects the Collector's local OpAMP connection. |

## Direct extension

From the repository root, use `scripts/opampextension-ws.yaml` or
`scripts/opampextension-http.yaml`. These examples target Commander on the Docker
host; replace `host.docker.internal` with `localhost` when running natively.

```sh
docker run --rm -v "$PWD/scripts/opampextension-ws.yaml:/config.yaml:ro" \
  ghcr.io/open-telemetry/opentelemetry-collector-releases/opentelemetry-collector-contrib:0.163.1 \
  --config=/config.yaml
```

On Linux Docker, add `--add-host=host.docker.internal:host-gateway` before the
image name. The examples use plaintext local-development endpoints; configure
`wss`/`https` and appropriate TLS settings for a secured deployment.

Set `extensions.opamp.instance_uid` to a unique persistent UUID for each
installation if identity must survive process restarts. This UID is independent
of the Collector resource's `service.instance.id`. Do not copy one UID across a
fleet. The extension does not support `agent_description.identifying_attributes`;
use the Collector resource and supported `non_identifying_attributes` instead.

`opampctl template agentremoteconfig opamp-extension` prints an AgentRemoteConfig
envelope containing a direct-reporting Collector example. Extract its `spec.value`
to use it as a local Collector configuration. Assigning that envelope to a directly
connected extension does not add remote-configuration support.

## Supervisor-managed Collector

Install Supervisor 0.163.0 and Contrib Collector 0.163.1 binaries from the
[official releases](https://github.com/open-telemetry/opentelemetry-collector-releases/releases).
Place them at `./opampsupervisor` and `./otelcol-contrib`, or adjust the executable
path in `scripts/opampsupervisor.yaml`. From the repository root, with Commander
listening on port 8080:

```sh
./otelcol-contrib validate --config=scripts/collector-supervised.yaml
./opampsupervisor --config=scripts/opampsupervisor.yaml
```

Keep `supervisor-data` persistent and separate per installed Supervisor. It stores
identity and configuration state. Do not run the directly connected Collector
example at the same time on the same OTLP ports.

Assign a pipeline configuration such as `opampctl template agentremoteconfig
otlp-debug` through an AgentGroup. Supervisor merges it with the local config and
injects its own OpAMP extension. Do not point the managed Collector's extension
directly at Commander. For HTTP transport, change only `server.endpoint` in the
Supervisor config to `http://localhost:8080/api/v1/opamp`.

## Connection-settings delivery and rotation

Commander requires both `AcceptsOpAMPConnectionSettings` and
`ReportsConnectionSettingsStatus` before sending a bundle containing an OpAMP
endpoint or certificate. Supervisor **0.163.0 is not supported for endpoint or
certificate rotation**: it replaces its client on each offer, loses the offer
hash, and does not report connection-settings status. Advertising acceptance
alone is insufficient. See [#850](https://github.com/minuk-dev/opampcommander/issues/850)
and the [upstream implementation](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.163.0/cmd/opampsupervisor/supervisor/supervisor.go).

For clients missing either capability, Commander withholds the **entire bundle**,
including any telemetry settings in that bundle. It keeps the desired settings
and hash intact; it does not synthesize an acknowledgment or mark the bundle
applied. Remove the OpAMP settings to deliver a telemetry-only bundle to such a
client. Supervisor own-metrics settings can reach the Collector without a hash
acknowledgment; verify effective configuration and health, while REST
`connectionSettings.syncStatus` remains `pending`.

For eligible clients, Commander continues offering the desired bundle, even after
an `APPLIED` acknowledgment. A saved acknowledgment may belong to a previous
client process; using it to suppress delivery would prevent a fresh client with
the same UID from recovering its settings. Clients deduplicate repeated offers
by hash while retaining their applied settings. After a restart that loses local
state, the client can receive and apply the bundle again; changed hashes also
remain deliverable. Sending a response alone never counts as application: REST
status reflects the last reported hash and status, not proof of the current
process's state. Clients should report lost state explicitly when possible.

There is **no supported Supervisor release for endpoint/certificate rotation**.
As of 2026-11-05, the 0.164.1 release and upstream `main` still lack Supervisor
connection-settings status reporting and offer-state restoration. Upstream
[#46157](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/46157)
describes the missing acknowledgment/persistence work, but its closure after
library changes is not proof that Supervisor implements it.
[#51015](https://github.com/open-telemetry/opentelemetry-collector-contrib/pull/51015)
preserves health across client replacement; it does not fix this delivery loop.
Keep remote rotation disabled and change Supervisor endpoint/TLS settings locally.

The required future release must advertise and correctly implement status
reporting, deduplicate offers across client replacement, recover settings after
process restart, and report success only after application. Enabling support
also requires real Supervisor endpoint replacement, uninterrupted Collector
health, repeated-offer idempotence and actual certificate rotation on both
WebSocket and HTTP. A reference-client mTLS test does not satisfy that gate.

### Regression coverage

`make test-e2e-supervisor-policy` runs released Supervisor **0.163.0** managing
Collector Contrib **0.163.1** in Docker, sequentially on **WSS and HTTPS**.
It deliberately enables `accepts_opamp_connection_settings` and verifies:

- Endpoint and certificate assignments remain pending without an invented ACK.
- Multiple subsequent reports do not replace the client or interrupt health.
- A mixed bundle is withheld in full, including its own-metrics destination.
- Restarting Supervisor with its original certificate and UID remains possible.
- Removing the unsupported OpAMP part unblocks telemetry, and two successive
  destinations reach the actual managed Collector after restart.

This runs in the dedicated `test-supervisor-policy` CI job. It brings the real
Supervisor scenario from [#846](https://github.com/minuk-dev/opampcommander/pull/846)
back into CI with the supported-client policy required by
[#850](https://github.com/minuk-dev/opampcommander/issues/850); it does not claim
that a withheld endpoint or certificate was rotated. The normal E2E suite also
counts reference-client OpAMP applications to verify identical offers apply
once, recovery after a fresh client starts, and delivery of a new revision.

## Capability option migration

- Since v0.163.0, `reports_remote_config: true` causes Supervisor startup to fail.
  Remove it: `accepts_remote_config` enables both acceptance and status reporting.
- Since v0.158.0, `reports_package_statuses` is removed. `accepts_packages` enables
  both package capabilities, but v0.163.0 still has unfinished package-state
  operations. The example leaves package updates disabled.
- Extension-only remote restart is separate from remote configuration. It requires
  `accepts_restart_command` and the `extension.opampextension.RemoteRestarts`
  feature gate; it reloads existing configuration sources and is not supported on
  Windows by this version.
- Optional `reports_raw_config` reports unexpanded source under `raw` alongside
  effective config. It may include inline secrets; leave it disabled unless the
  server should receive those contents.

See the [upstream extension documentation](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.163.0/extension/opampextension/README.md)
and [Supervisor documentation](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.163.0/cmd/opampsupervisor/README.md).
