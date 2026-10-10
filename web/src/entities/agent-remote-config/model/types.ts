import type { AgentRemoteConfigSpec, Attributes, Condition } from '@shared/api';

export interface AgentRemoteConfigMetadata {
  resourceVersion?: string;
  name: string;
  namespace: string;
  attributes?: Attributes;
  createdAt: string;
}

export interface AgentRemoteConfig {
  metadata: AgentRemoteConfigMetadata;
  spec: AgentRemoteConfigSpec;
  status?: { conditions?: Condition[]; schemaRefsSource?: 'auto' | 'explicit' };
}
