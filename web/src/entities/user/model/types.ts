import type { Condition } from '@shared/api';
import type { Role } from '@entities/role';
import type { RoleBinding } from '@entities/role-binding';

export interface UserMetadata {
  uid: string;
  createdAt: string;
  updatedAt: string;
  deletedAt?: string | null;
  labels?: Record<string, string>;
}

export interface UserSpec {
  email: string;
  username: string;
  isActive: boolean;
  // Write-only: plaintext password for basic (username/password) auth. Sent only when creating a
  // user; the server stores only a one-way hash and never returns it, so responses omit this field.
  password?: string;
}

export interface User {
  kind: string;
  apiVersion: string;
  metadata: UserMetadata;
  spec: UserSpec;
  status?: { conditions?: Condition[]; roles?: string[] };
}

export interface UserRoleEntry {
  role: Role;
  roleBinding?: RoleBinding | null;
}

export interface UserProfileResponse {
  user: User;
  roles?: UserRoleEntry[];
}
