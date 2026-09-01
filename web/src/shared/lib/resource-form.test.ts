import { describe, expect, it } from 'vitest';
import { parseAttributes, validateResourceName } from './resource-form';

describe('validateResourceName', () => {
  it('accepts an ordinary resource name', () => {
    expect(validateResourceName('otlp-debug')).toBeNull();
  });

  it('rejects what would break the resource URL', () => {
    expect(validateResourceName('')).toMatch(/required/);
    expect(validateResourceName('  ')).toMatch(/required/);
    expect(validateResourceName('two words')).toMatch(/whitespace/);
    expect(validateResourceName('a/b')).toMatch(/"\/"/);
  });
});

describe('parseAttributes', () => {
  it('reads a YAML map', () => {
    expect(parseAttributes('team: platform\ntier: prod')).toEqual({
      team: 'platform',
      tier: 'prod',
    });
  });

  it('treats an empty buffer as no attributes', () => {
    expect(parseAttributes('   ')).toEqual({});
  });

  it('stringifies non-string values rather than sending numbers', () => {
    expect(parseAttributes('replicas: 3\nenabled: true')).toEqual({
      replicas: '3',
      enabled: 'true',
    });
  });

  it('rejects a buffer that is not a map', () => {
    expect(() => parseAttributes('- a\n- b')).toThrow(/map/);
  });
});
