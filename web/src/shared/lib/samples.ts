// Loads sample payloads for the YAML/JSON editor dialogs.
//
// The actual samples live as plain YAML files under web/public/samples/ so
// anyone (no TS knowledge required) can add, tweak, or remove them. See
// web/public/samples/README.md for the file format.

import type { CodeSample } from '@shared/ui';
import { fromYAML } from './yaml';

export type SamplesPath = `/samples/${string}.yaml`;

function interpolate(text: string, vars: Record<string, string>): string {
  return text.replace(/\{\{(\w+)\}\}/g, (match, key) => {
    if (!Object.prototype.hasOwnProperty.call(vars, key)) return match;
    const v = vars[key];
    if (typeof v !== 'string') return match;
    return v;
  });
}

// Fetches a YAML sample file, interpolates `{{var}}` placeholders, and parses it.
// Exported so domain-specific loaders (e.g. agent groups) can reuse the fetch +
// interpolate pipeline without duplicating it.
export async function fetchYaml<T>(path: SamplesPath, vars: Record<string, string>): Promise<T> {
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
