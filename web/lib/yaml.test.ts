import { describe, expect, it } from 'vitest';
import { fromYAML, toYAML } from './yaml';

describe('yaml', () => {
  it('round-trips primitives and arrays', () => {
    const cases: unknown[] = [
      null,
      true,
      0,
      42,
      'hello',
      ['a', 'b', 'c'],
      { name: 'foo', count: 3 },
      { nested: { list: [1, 2, 3], flag: false } },
    ];
    for (const original of cases) {
      const text = toYAML(original);
      expect(fromYAML(text)).toEqual(original);
    }
  });

  it('round-trips UUID and timestamp strings as plain strings (not numbers)', () => {
    const value = {
      id: '4d1ff377-260f-453a-8e21-c8fcd690bf4a',
      ts: '2026-05-22T13:45:00Z',
    };
    expect(fromYAML(toYAML(value))).toEqual(value);
  });

  it('round-trips quoted-number-like strings without losing the type', () => {
    // "0.122.0" looks number-ish but must come back as a string.
    const value = { version: '0.122.0' };
    const round = fromYAML(toYAML(value));
    expect(round).toEqual(value);
    expect(typeof (round as { version: unknown }).version).toBe('string');
  });

  it('emits empty containers compactly', () => {
    expect(toYAML({})).toBe('{}\n');
    expect(toYAML([])).toBe('[]\n');
  });

  it('preserves dotted keys without quoting them', () => {
    const value = { 'service.name': 'otelcol' };
    const text = toYAML(value);
    expect(text).toContain('service.name: otelcol');
    expect(fromYAML(text)).toEqual(value);
  });

  it('handles multiline strings via block scalar', () => {
    const value = { body: 'line1\nline2\nline3' };
    const round = fromYAML(toYAML(value));
    expect(round).toEqual(value);
  });

  it('returns empty string for undefined input', () => {
    expect(toYAML(undefined)).toBe('');
  });
});
