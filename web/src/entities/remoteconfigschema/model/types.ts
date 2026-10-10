import type { Attributes, Condition } from '@shared/api';

export interface ConfigField {
  type?: string;
  children?: Record<string, ConfigField>;
  open?: boolean;
  enum?: string[];
  doc?: string;
}

export interface Component {
  type: string;
  signals?: string[];
  stability?: Record<string, string>;
  pairs?: { from: string; to: string }[];
  module?: string;
  fields?: ConfigField;
}

export interface RemoteConfigSchema {
  kind?: string;
  apiVersion?: string;
  metadata: { name: string; namespace: string; attributes?: Attributes; createdAt: string };
  spec: {
    binary: string;
    version: string;
    components: Record<string, Record<string, Component>>;
  };
  status?: { conditions?: Condition[] };
}
