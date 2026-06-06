import { describe, expect, it } from 'vitest';
import { LOCAL_TIME_ZONE, isValidTimeZone, listTimeZones } from './preferences';

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
