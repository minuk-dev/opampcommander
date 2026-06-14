import type { Condition } from '@shared/api';

export interface RoleBindingMetadata {
  namespace: string;
  name: string;
  createdAt?: string;
  updatedAt?: string;
  deletedAt?: string | null;
}

export interface RoleBindingRoleRef {
  kind: string;
  name: string;
}

export interface RoleBindingSubject {
  kind: string;
  name: string;
  apiVersion?: string;
}

export interface RoleBindingSpec {
  roleRef: RoleBindingRoleRef;
  subjects?: RoleBindingSubject[];
}

export interface RoleBinding {
  kind: string;
  apiVersion: string;
  metadata: RoleBindingMetadata;
  spec: RoleBindingSpec;
  status?: { conditions?: Condition[] };
}
