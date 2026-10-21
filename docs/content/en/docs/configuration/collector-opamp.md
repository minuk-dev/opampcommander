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
