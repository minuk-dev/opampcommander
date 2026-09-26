import type { Condition } from '@shared/api';

export interface Application {
  kind: string;
  apiVersion: string;
  metadata: {
    id: string;
    name: string;
    labels?: Record<string, string>;
    annotations?: Record<string, string>;
    firstSeenAt: string;
    lastSeenAt: string;
  };
  spec: {
    namespace?: string;
    name: string;
    versions?: string[];
    agentType?: string;
  };
  status: { agentInstanceUids: string[]; conditions?: Condition[] };
}
