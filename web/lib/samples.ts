// Loads sample payloads for the YAML/JSON editor dialogs.
//
// The actual samples live as plain YAML files under web/public/samples/ so
// anyone (no TS knowledge required) can add, tweak, or remove them. See
// web/public/samples/README.md for the file format.

import type { CodeSample } from '@/components/CodeEditorDialog';
import type { AgentGroupSpec } from '@/lib/types';
import { fromYAML } from '@/lib/yaml';

export type SamplesPath = `/samples/${string}.yaml`;

function interpolate(text: string, vars: Record<string, string>): string {
  return text.replace(/\{\{(\w+)\}\}/g, (_, key) =>
    Object.prototype.hasOwnProperty.call(vars, key) ? vars[key] : `{{${key}}}`,
  );
}

async function fetchYaml<T>(path: SamplesPath, vars: Record<string, string>): Promise<T> {
  const res = await fetch(path);
  if (!res.ok) {
    throw new Error(`Failed to load ${path}: ${res.status} ${res.statusText}`);
  }
  const text = await res.text();
  const merged = { now: new Date().toISOString(), ...vars };
  return fromYAML(interpolate(text, merged)) as T;
}

export async function loadSamples(
  path: SamplesPath,
  vars: Record<string, string> = {},
): Promise<CodeSample[]> {
  const data = await fetchYaml<unknown>(path, vars);
  if (!Array.isArray(data)) {
    throw new Error(`Expected ${path} to be a YAML list of samples`);
  }
  return data as CodeSample[];
}

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
