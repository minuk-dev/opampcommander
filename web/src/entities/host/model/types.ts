import type { Condition } from '@shared/api';

export interface HostMetadata {
  id: string;
  name?: string;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  firstSeenAt: string;
  lastSeenAt: string;
}

export interface HostSpec {
  /** Deployment environment: baremetal | vm | docker | kubernetes | ecs | unknown. */
  platform: string;
  arch?: string;
  type?: string;
  osType?: string;
  osVersion?: string;
  cloudProvider?: string;
  cloudPlatform?: string;
  cloudRegion?: string;
}

export interface HostStatus {
  agentInstanceUids: string[];
  conditions?: Condition[];
}

export interface Host {
  kind: string;
  apiVersion: string;
  metadata: HostMetadata;
  spec: HostSpec;
  status: HostStatus;
}
