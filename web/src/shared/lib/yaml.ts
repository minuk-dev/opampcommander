// Thin wrappers over js-yaml so the rest of the app talks to one place.
// Serialization tweaks: 2-space indent, no inline flow collections, keep keys
// in insertion order.

import * as yaml from 'js-yaml';

export function toYAML(value: unknown): string {
  if (value === undefined) return '';
  try {
    return yaml.dump(value, {
      indent: 2,
      lineWidth: 120,
      noRefs: true,
      sortKeys: false,
      // -1 disables block scalar truncation so long strings keep flowing.
      flowLevel: -1,
    });
  } catch (err) {
    return `# yaml dump failed: ${err instanceof Error ? err.message : String(err)}\n`;
  }
}

export function fromYAML(text: string): unknown {
  return yaml.load(text, {
    // YAML 1.1 types and merge keys — collector configs use both, and the
    // stricter CORE_SCHEMA js-yaml defaults to since v5 rejects them.
    // Duplicate keys throw, since `json` stays off.
    schema: yaml.YAML11_SCHEMA,
  });
}
