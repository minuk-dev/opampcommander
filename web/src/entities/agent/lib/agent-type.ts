// Helpers for the agent `metadata.type` classification (the OpenTelemetry
// Collector distribution derived from `service.name`). Shared by the agents list
// and detail pages so the "is this a Collector?" rule lives in one place.

/** The server's sentinel for an unclassified agent. */
export const AGENT_TYPE_UNKNOWN = 'unknown';

/**
 * Whether the agent is an OpenTelemetry Collector of any distribution. Mirrors
 * the server-side `(Type).IsOTelCollector()`: any non-empty, non-`unknown` type
 * follows the `otelcol`/`otelcol-<distro>` naming convention.
 */
export function isOtelCollector(type: string | undefined): boolean {
  return Boolean(type) && type !== AGENT_TYPE_UNKNOWN;
}

/** Display label for an agent type, falling back to `unknown` when absent. */
export function agentTypeLabel(type: string | undefined): string {
  return type || AGENT_TYPE_UNKNOWN;
}
