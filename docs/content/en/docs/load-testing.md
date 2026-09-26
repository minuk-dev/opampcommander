---
title: Load testing and capacity SLOs
---

`cmd/loadtest` connects real OpAMP WebSocket clients to a running server. It
ramps connections, reports health, optionally replaces agents, and updates one
AgentGroup's remote config. The JSON result records connect and config-offer
latency percentiles, errors, server metrics at the start/end, and sampled peaks.
It removes the AgentGroup when the run ends. Use a disposable namespace or test server:
agents created by a run remain in persistence after they disconnect.

```sh
go run ./cmd/apiserver/main.go --config ./configs/apiserver/standalone.yaml
# In another terminal:
go run ./cmd/loadtest -count 1000 -rate 50 -duration 10m \
  -report-interval 30s -churn-interval 1s -ops-interval 1m \
  -username admin -password admin -output result.json
```

For a remote server, set `-url`, `-api-url`, and `-metrics-url`. Use
`-token`/`OPAMP_LOAD_TOKEN` for management operations; `-username` and
`-password` are convenient for local standalone runs. Set `-mongo-uri` to
capture MongoDB `serverStatus.opcounters` before and after the test. The
`mongo_per_second` values include unrelated
traffic on the same MongoDB server. A failed metrics scrape is reported in
`metrics_error`; missing data must not be interpreted as zero. The command
does not inject REST read traffic, restart nodes, or create 100,000 groups.

The scheduled GitHub Action runs 50 agents for 90 seconds against the in-memory
standalone server and uploads the report and server log. This catches protocol
and harness regressions. It is not a capacity claim: use MongoDB, the intended
Kafka/direct topology, representative data and dedicated hardware for scale
tests. Record the server commit, machine and container limits, MongoDB topology,
scenario flags, data seed, connected/disconnected mix, and all result artifacts.

## SLO measurement contract

Agree numeric budgets after a production-like baseline; no 300,000-agent
capacity has been demonstrated. Measure p50/p95/p99 for REST list/get, first
CLI row, agent connect, remote-config offer and effective-config acknowledgement.
Also record error rate, peak server memory and goroutines, MongoDB operation
rates, documents examined, queue age, and reconciliation completion time.
`propagation_latency` here ends when the agent sees an offer; effective-config
persistence and group reconciliation need separate server-side measurements.
`connection_events` includes reconnects and is not a live-connection gauge.

At each scale, vary namespace skew, selector overlap, config size and offline
fraction. Run reads alongside heartbeat/discovery, group edits, rotation,
reconnect bursts, rolling restarts and namespace deletion. The large-fleet
targets (300,000 agents, 50 namespaces, 100,000 groups) require an explicitly
provisioned environment. Capture MongoDB profiler/explain output for queries
and documents examined. For operator runs, measure initial list/cache size,
status-write rate, workqueue convergence and leader failover; avoid mirroring
per-agent heartbeats into Agent CRs.

## Small local baseline

On 2026-09-27, a local macOS arm64 host ran server commit `d49bd3aa2` with
Go 1.27.1, in-memory persistence and one server. A 10-agent, 10/s, 15-second
run with 3-second health reports and 5-second group pushes produced 10
connections, zero connection/server errors, two successful pushes, connect
p50/p95/p99 of 1.39/1.77/1.77 ms, and observed offer p50/p95/p99 of
1002/1002/1002 ms (11 observations out of up to 20 offers). Server metrics were scraped, but this
host had already been running other workloads, so their absolute counters
are excluded. This is a smoke baseline, not an SLO or capacity estimate.

The included [Grafana dashboard](/loadtest-dashboard.json) shows the
server's runtime footprint and liveness write rate. The JSON report is the
source for client-side latency. MongoDB and Kafka dashboards need exporter
metrics from the actual deployment.
