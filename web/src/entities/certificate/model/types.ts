import type { Attributes, Condition } from '@shared/api';

export interface CertificateMetadata {
  name: string;
  namespace: string;
  attributes?: Attributes;
  createdAt: string;
  deletedAt?: string | null;
}

export interface CertificateSpec {
  cert?: string;
  privateKey?: string;
  caCert?: string;
}

export interface Certificate {
  kind: string;
  apiVersion: string;
  metadata: CertificateMetadata;
  spec: CertificateSpec;
  status?: { conditions?: Condition[] };
}
