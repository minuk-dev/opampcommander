// Shared structural types mirroring api/v1/*.go on the server. These are the
// cross-cutting envelope/value types reused by multiple entities; entity-specific
// types live under each entity's model/.

export interface ListMeta {
  continue: string;
  remainingItemCount: number;
}

export interface ListResponse<T> {
  kind: string;
  apiVersion: string;
  metadata: ListMeta;
  items: T[];
}

export interface Condition {
  type: string;
  lastTransitionTime: string;
  status: 'True' | 'False' | 'Unknown';
  reason: string;
  message?: string;
}

// Generic attribute bag shared by agent groups, certificates, packages, etc.
export type Attributes = Record<string, string>;

// ---------- Connection settings (shared by agent spec + agent group config) ----------
export interface OpAMPConnectionSettings {
  destinationEndpoint: string;
  headers?: Record<string, string[]>;
  certificateName?: string | null;
}

export interface TelemetryConnectionSettings {
  destinationEndpoint: string;
  headers?: Record<string, string[]>;
  certificateName?: string | null;
}

export interface OtherConnectionSettings {
  destinationEndpoint: string;
  headers?: Record<string, string[]>;
  certificateName?: string | null;
}

export interface ConnectionSettings {
  opamp?: OpAMPConnectionSettings;
  ownMetrics?: TelemetryConnectionSettings;
  ownLogs?: TelemetryConnectionSettings;
  ownTraces?: TelemetryConnectionSettings;
  otherConnections?: Record<string, OtherConnectionSettings>;
}

// Remote-config spec shared by AgentGroup config and the AgentRemoteConfig entity.
export interface AgentRemoteConfigSpec {
  value: string;
  contentType: string;
}

// Telemetry signals an Endpoint (or one of its tenants) supports.
export interface EndpointSignals {
  metrics: boolean;
  logs: boolean;
  traces: boolean;
}

// A tenant of an Endpoint: lets the same destination be managed differently per
// tenant via per-tenant headers (e.g. X-Scope-OrgID) and tags.
export interface EndpointTenant {
  name: string;
  headers?: Record<string, string>;
  tags?: Record<string, string>;
  signals?: EndpointSignals;
}

// Spec for the Endpoint entity (a telemetry backend/destination).
export interface EndpointSpec {
  url: string;
  protocol: string;
  signals: EndpointSignals;
  tenants?: EndpointTenant[];
}
