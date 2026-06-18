import type { Attributes, Condition, EndpointSpec } from '@shared/api';

export interface EndpointMetadata {
  name: string;
  namespace: string;
  attributes?: Attributes;
  createdAt: string;
}

export interface Endpoint {
  metadata: EndpointMetadata;
  spec: EndpointSpec;
  status?: { conditions?: Condition[] };
}
