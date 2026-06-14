import type { Condition } from '@shared/api';

export interface NamespaceMetadata {
  name: string;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  createdAt: string;
  deletedAt?: string | null;
}

export interface NamespaceStatus {
  conditions?: Condition[];
}

export interface Namespace {
  metadata: NamespaceMetadata;
  status: NamespaceStatus;
}
