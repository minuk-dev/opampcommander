import type { AgentRemoteConfigSpec, Attributes, Condition, ConnectionSettings } from '@shared/api';

export interface AgentSelector {
  identifyingAttributes?: Record<string, string>;
  nonIdentifyingAttributes?: Record<string, string>;
}

export interface AgentGroupRemoteConfig {
  agentRemoteConfigName?: string;
  agentRemoteConfigSpec?: AgentRemoteConfigSpec;
  agentRemoteConfigRef?: string;
}

export interface AgentGroupAgentConfig {
  agentRemoteConfig?: AgentGroupRemoteConfig;
  connectionSettings?: ConnectionSettings;
}

export interface AgentGroupMetadata {
  namespace: string;
  name: string;
  attributes: Attributes;
  createdAt: string;
  deletedAt?: string | null;
}

export interface AgentGroupSpec {
  priority: number;
  selector: AgentSelector;
  agentConfig?: AgentGroupAgentConfig;
}

export interface AgentGroupStatus {
  numAgents: number;
  numConnectedAgents: number;
  numHealthyAgents: number;
  numUnhealthyAgents: number;
  numNotConnectedAgents: number;
  conditions?: Condition[];
}

export interface AgentGroup {
  metadata: AgentGroupMetadata;
  spec: AgentGroupSpec;
  status: AgentGroupStatus;
}
