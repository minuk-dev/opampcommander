// Helpers shared by the resource create/edit dialogs. These concern the generic
// resource envelope (name, attributes) rather than any one domain type.

import type { Attributes } from '@shared/api';
import { fromYAML } from './yaml';

// validateResourceName rejects only what would break the resource's URL path;
// the server remains the authority on everything else.
export function validateResourceName(name: string): string | null {
  if (name.trim() === '') return 'Name is required.';
  if (/\s/.test(name)) return 'Name must not contain whitespace.';
  if (name.includes('/')) return 'Name must not contain "/".';
  return null;
}

// parseAttributes reads an attributes editor buffer as a YAML string map.
export function parseAttributes(text: string): Attributes {
  if (text.trim() === '') return {};
  const parsed = fromYAML(text);
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    throw new Error('attributes must be a YAML map');
  }
  return Object.fromEntries(
    Object.entries(parsed as Record<string, unknown>).map(([k, v]) => [k, String(v)]),
  );
}
