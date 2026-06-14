import type { Condition, ConnectionSettings } from '@shared/api';

export interface AgentDescription {
  identifyingAttributes?: Record<string, string>;
  nonIdentifyingAttributes?: Record<string, string>;
}

export interface AgentCustomCapabilities {
  capabilities?: string[];
}

export interface AgentMetadata {
  instanceUid: string;
  namespace: string;
  description?: AgentDescription;
  capabilities?: number;
  customCapabilities?: AgentCustomCapabilities;
}

export interface AgentSpecRemoteConfig {
  remoteConfigNames?: string[];
}

export interface AgentSpecPackages {
  packages?: string[];
}

export interface AgentSpec {
  newInstanceUid?: string;
  connectionSettings?: ConnectionSettings;
  remoteConfig?: AgentSpecRemoteConfig;
  packagesAvailable?: AgentSpecPackages;
  restartRequiredAt?: string | null;
}

export interface AgentConfigFile {
  body: string;
  contentType: string;
}

export interface AgentConfigMap {
  configMap?: Record<string, AgentConfigFile>;
}

export interface AgentEffectiveConfig {
  configMap: AgentConfigMap;
}

export interface AgentComponentHealth {
  healthy: boolean;
  startTime?: string;
  lastError?: string;
  status?: string;
  statusTime?: string;
  componentsMap?: Record<string, string>;
}

export interface AgentStatus {
  effectiveConfig?: AgentEffectiveConfig;
  packageStatuses?: unknown;
  componentHealth: AgentComponentHealth;
  availableComponents?: unknown;
  conditions?: Condition[];
  connected: boolean;
  connectionType?: string;
  sequenceNum?: number;
  lastReportedAt?: string;
}

export interface Agent {
  metadata: AgentMetadata;
  spec?: AgentSpec;
  status: AgentStatus;
}
