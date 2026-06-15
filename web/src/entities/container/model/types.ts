import type { Condition } from '@shared/api';

export interface ContainerMetadata {
  id: string;
  name?: string;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  firstSeenAt: string;
  lastSeenAt: string;
}

export interface ContainerSpec {
  /** Deployment environment: docker | kubernetes | ecs | unknown. */
  platform: string;
  imageName?: string;
  runtime?: string;
  /** The host (node) this container runs on, when known. */
  hostId?: string;
  k8sPodName?: string;
  k8sNamespaceName?: string;
  k8sNodeName?: string;
}

export interface ContainerStatus {
  agentInstanceUids: string[];
  conditions?: Condition[];
}

export interface Container {
  kind: string;
  apiVersion: string;
  metadata: ContainerMetadata;
  spec: ContainerSpec;
  status: ContainerStatus;
}
