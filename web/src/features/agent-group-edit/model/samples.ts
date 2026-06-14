import type { AgentGroupSpec } from '@entities/agent-group';
import { fetchYaml } from '@shared/lib';

// AgentGroup samples carry extra top-level fields (name, attributes, spec)
// because the AgentGroup editor splits them into separate text areas.
export interface AgentGroupSample {
  label: string;
  description?: string;
  name: string;
  attributes: Record<string, string>;
  spec: AgentGroupSpec;
}

export async function loadAgentGroupSamples(): Promise<AgentGroupSample[]> {
  const data = await fetchYaml<unknown>('/samples/agentgroups.yaml', {});
  if (!Array.isArray(data)) {
    throw new Error('Expected /samples/agentgroups.yaml to be a YAML list');
  }
  return data as AgentGroupSample[];
}
