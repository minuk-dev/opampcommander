---
title: "Kafka outages"
weight: 10
type: docs
description: >
  Delivery guarantees, recovery and signals during Kafka event backend outages.
---

Kafka events are best-effort notifications of state already saved in the database.
An API success confirms persistence, not broker delivery or an agent acknowledgement.
The same-node event path dispatches directly to local connections without Kafka.

For remote agents, notifications normally prompt an immediate push. If a notification
or cache invalidation is lost, subsequent OpAMP heartbeats reload persisted state
once the agent cache expires (30s by default). Cache reads do not extend that TTL.
Recovery therefore depends on the database remaining available and the agent sending
another message; it does not require Kafka to reconnect. Later updates may supersede
older desired state. This is state convergence, not an ordered replay of every command
or an exactly-once execution guarantee.

## Limits and recovery

`event.kafka.sendTimeout` (default 2s) bounds submission to Sarama and individual
producer/socket operations. `retryAttempts` (default 2, including the initial attempt)
and `retryBackoff` (default 100ms) configure Sarama's native retries. Metadata retries
have a separate budget; `sendTimeout` does not cap the whole background retry cycle.
The producer uses Sarama's own buffers. Exhausted deliveries are logged and counted;
there is no application replay queue, DLQ or custom circuit breaker. Notification
workers and enqueue waits are bounded so broker acknowledgements do not hold API
request goroutines. Producer shutdown drains both result channels with a bounded wait.

A broker outage after startup leaves REST and locally connected agents serving.
Kafka is still required during Kafka client construction: starting a new server while
all brokers are unavailable can fail. The degradation guarantee applies to an already
running server.

## Operator signals

Enable management Prometheus metrics to export:

| Instrument | Meaning |
|---|---|
| `opampcommander.kafka.events`, `outcome=enqueued` | Submitted to Sarama; not a delivery acknowledgement |
| Same counter, `outcome=delivered` | Acknowledged by Kafka; not by the target agent |
| Same counter, `outcome=failed` | Terminal producer failures after retries; notification dropped |
| Same counter, `outcome=rejected` | Encoding, cancellation, timeout or closed-producer error before enqueue |
| `opampcommander.kafka.delivery_degraded` | 1 if the last observed producer result failed; otherwise 0 |

Prometheus exposes dots as underscores and appends `_total` to counters, for example
`opampcommander_kafka_events_total{outcome="failed"}`. Counters are per process and
reset on restart. Outcomes are stages rather than mutually exclusive totals: an
enqueued notification can subsequently be delivered or fail. No agent or server ID
labels are added.

`/healthz` includes a `KafkaProducer` reason after a terminal producer failure, and
clears it after the next observed successful delivery. Both `/healthz` and `/readyz`
remain HTTP 200 for this degradation: draining every replica would interrupt working
local connections. Encoding errors and caller cancellations are counted as rejected,
not treated as broker failures. Queue acceptance alone never clears a delivery failure.

These signals describe the last **producer result**, not current cluster reachability.
Before any result, the gauge is 0; mixed partition results can clear it even if another
partition remains unavailable. Idle broker outages and consumer-only failures need
an active, bounded, cached broker probe tracked in
[issue #595](https://github.com/minuk-dev/opampcommander/issues/595).

## Issue #592 closure criteria

[PR #735](https://github.com/minuk-dev/opampcommander/pull/735) already supplied native
retry/backoff, bounded asynchronous enqueue and heartbeat/cache-expiry recovery. It
intentionally removed the custom breaker, application replay queue and unreachable
HTTP error from its earlier implementation. Its description used `Refs #592`, so its
merge did not close [issue #592](https://github.com/minuk-dev/opampcommander/issues/592).

The original checklist still mixes that earlier delivery proposal with the merged
best-effort contract:

| Original task | Current implementation / remaining decision |
|---|---|
| Retry/backoff and circuit breaker | Native retries and bounded enqueue exist; a custom breaker was intentionally removed in #735 |
| Local fallback | Direct same-node dispatch already bypasses Kafka |
| Replay buffer or DLQ | Persisted state plus heartbeat/cache expiry supplies convergence; ordered command replay is not implemented |
| Distinct unreachable error / HTTP status | Persisted updates succeed; asynchronous notification failure cannot truthfully change an already returned HTTP response |
| Sent/failed/dropped and breaker metrics | Enqueued/delivered/terminal-failed/rejected counters and last-result degradation are available; there is no breaker to measure |
| Broker killed mid-flight | `TestE2E_APIServer_KafkaOutage` stops the broker, checks actual config application on local and remote reference agents while it stays down, then checks producer recovery on restart |
| Broker health | Last producer outcome is exposed; active reachability/consumer monitoring remains #595 |

To close #592 under the merged architecture, accept state convergence as the recovery
contract in place of ordered replay, the bounded producer as the outage isolation
mechanism, and successful persistence as the API contract; require the outage E2E to
pass, and explicitly track active broker health in #595. If the original breaker,
DLQ/replay and unreachable HTTP semantics remain requirements, those are still missing
and #592 must remain open. This change does not close the issue automatically.
