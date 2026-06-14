import type { Attributes, Condition } from '@shared/api';

export interface AgentPackageMetadata {
  name: string;
  namespace: string;
  attributes?: Attributes;
  createdAt: string;
  deletedAt?: string | null;
}

export interface AgentPackageSpec {
  packageType: string;
  version: string;
  downloadUrl: string;
  contentHash?: string;
  signature?: string;
  headers?: Record<string, string>;
  hash?: string;
}

export interface AgentPackage {
  metadata: AgentPackageMetadata;
  spec: AgentPackageSpec;
  status?: { conditions?: Condition[] };
}
