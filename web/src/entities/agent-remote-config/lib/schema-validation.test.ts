import { describe, expect, it } from 'vitest';
import {
  schemaRefsSource,
  skipsSchemaValidation,
  SKIP_SCHEMA_VALIDATION,
} from './schema-validation';
import type { AgentRemoteConfig } from '../model/types';

const config: AgentRemoteConfig = {
  metadata: { name: 'cfg', namespace: 'default', createdAt: '' },
  spec: { value: '', contentType: 'text/yaml', schemaRefs: ['schema'] },
};

describe('schema validation metadata', () => {
  it('keeps legacy provenance unknown and distinguishes recorded sources', () => {
    expect(schemaRefsSource(config)).toBe('Source unavailable (existing config)');
    expect(schemaRefsSource({ ...config, status: {} })).toBe(
      'Source unavailable (existing config)',
    );
    for (const [source, label] of [
      ['auto', 'Auto-resolved'],
      ['explicit', 'Explicitly set'],
    ] as const) {
      expect(
        schemaRefsSource({
          ...config,
          status: { schemaRefsSource: source },
        }),
      ).toBe(label);
    }
    expect(
      schemaRefsSource({
        ...config,
        spec: { ...config.spec, schemaRefs: [] },
        status: { schemaRefsSource: 'explicit' },
      }),
    ).toBe('No schema pinned');
  });
  it('matches the backend boolean annotation semantics', () => {
    for (const value of ['1', 't', 'T', 'true', 'True', 'TRUE', 'false', '0', 'yes', '']) {
      expect(
        skipsSchemaValidation({
          ...config,
          metadata: { ...config.metadata, attributes: { [SKIP_SCHEMA_VALIDATION]: value } },
        }),
      ).toBe(['1', 't', 'T', 'true', 'True', 'TRUE'].includes(value));
    }
  });
});
