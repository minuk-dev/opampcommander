import type { Condition } from '@shared/api';

export interface RoleMetadata {
  uid: string;
  createdAt: string;
  updatedAt: string;
  deletedAt?: string | null;
}

export interface RoleSpec {
  displayName: string;
  description: string;
  permissions?: string[];
  isBuiltIn: boolean;
}

export interface Role {
  kind: string;
  apiVersion: string;
  metadata: RoleMetadata;
  spec: RoleSpec;
  status?: { conditions?: Condition[] };
}
