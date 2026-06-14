import { describe, expect, it } from 'vitest';
import {
  ABSOLUTE_TIME_FORMAT,
  LOCAL_TIME_ZONE,
  RELATIVE_TIME_FORMAT,
  canonicalTimeFormat,
  isValidTimeZone,
  listTimeZones,
} from './preferences';

describe('isValidTimeZone', () => {
  it('accepts the local sentinel and real IANA zones', () => {
    expect(isValidTimeZone(LOCAL_TIME_ZONE)).toBe(true);
    expect(isValidTimeZone('UTC')).toBe(true);
    expect(isValidTimeZone('Asia/Seoul')).toBe(true);
    expect(isValidTimeZone('America/New_York')).toBe(true);
  });

  it('rejects unknown zones and non-strings', () => {
    expect(isValidTimeZone('Mars/Olympus')).toBe(false);
    expect(isValidTimeZone('')).toBe(false);
    expect(isValidTimeZone(undefined)).toBe(false);
    expect(isValidTimeZone(42)).toBe(false);
  });
});

describe('listTimeZones', () => {
  it('returns a non-empty list that includes UTC', () => {
    const zones = listTimeZones();
    expect(zones.length).toBeGreaterThan(0);
    expect(zones).toContain('UTC');
  });
});

describe('canonicalTimeFormat', () => {
  it('keeps the explicit absolute opt-in', () => {
    expect(canonicalTimeFormat(ABSOLUTE_TIME_FORMAT)).toBe(ABSOLUTE_TIME_FORMAT);
  });

  it('defaults anything else to relative', () => {
    expect(canonicalTimeFormat(RELATIVE_TIME_FORMAT)).toBe(RELATIVE_TIME_FORMAT);
    expect(canonicalTimeFormat(undefined)).toBe(RELATIVE_TIME_FORMAT);
    expect(canonicalTimeFormat('bogus')).toBe(RELATIVE_TIME_FORMAT);
    expect(canonicalTimeFormat(42)).toBe(RELATIVE_TIME_FORMAT);
  });
});
