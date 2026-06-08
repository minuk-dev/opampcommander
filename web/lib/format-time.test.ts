import { describe, expect, it } from 'vitest';
import { formatRelativeTime, formatTimestamp } from './format-time';

describe('formatTimestamp', () => {
  const iso = '2026-06-04T01:59:30Z';

  it('returns null for empty or invalid input', () => {
    expect(formatTimestamp('', 'UTC')).toBeNull();
    expect(formatTimestamp(null, 'UTC')).toBeNull();
    expect(formatTimestamp(undefined)).toBeNull();
    expect(formatTimestamp('not-a-date', 'UTC')).toBeNull();
  });

  it('formats an explicit UTC zone', () => {
    const out = formatTimestamp(iso, 'UTC');
    expect(out).not.toBeNull();
    expect(out!.text).toBe('2026-06-04 01:59:30 UTC');
    // The title uses the long zone name and strips all commas.
    expect(out!.title).toContain('Coordinated Universal Time');
    expect(out!.title).not.toContain(',');
  });

  it('honours an explicit IANA timezone', () => {
    const out = formatTimestamp(iso, 'Asia/Seoul');
    expect(out).not.toBeNull();
    // Seoul is UTC+9, so 01:59:30Z is 10:59:30 local.
    expect(out!.text).toBe('2026-06-04 10:59:30 GMT+9');
  });

  it('formats a different zone correctly', () => {
    const out = formatTimestamp(iso, 'America/New_York');
    // EDT is UTC-4 in June, so 01:59:30Z is the previous day 21:59:30.
    expect(out!.text).toBe('2026-06-03 21:59:30 EDT');
  });
});

describe('formatRelativeTime', () => {
  const now = Date.parse('2026-06-04T12:00:00Z');
  const at = (offsetMs: number) => new Date(now + offsetMs).toISOString();
  const SEC = 1000;
  const MIN = 60 * SEC;
  const HOUR = 60 * MIN;
  const DAY = 24 * HOUR;

  it('returns null for empty or invalid input', () => {
    expect(formatRelativeTime('', now)).toBeNull();
    expect(formatRelativeTime(null, now)).toBeNull();
    expect(formatRelativeTime(undefined, now)).toBeNull();
    expect(formatRelativeTime('not-a-date', now)).toBeNull();
  });

  it('renders the present moment as "now"', () => {
    expect(formatRelativeTime(at(0), now)).toBe('now');
  });

  it('treats small future skew as "now" but keeps genuine future times', () => {
    // Within the clock-skew tolerance → "now" rather than "in 3 seconds".
    expect(formatRelativeTime(at(3 * SEC), now)).toBe('now');
    // Beyond the tolerance → still a real future phrase.
    expect(formatRelativeTime(at(20 * SEC), now)).toBe('in 20 seconds');
  });

  it('rolls a near-boundary value up to the next unit', () => {
    // 59.6s must read "1 minute ago", not "60 seconds ago".
    expect(formatRelativeTime(at(-59_600), now)).toBe('1 minute ago');
  });

  it('renders recent past times', () => {
    expect(formatRelativeTime(at(-30 * SEC), now)).toBe('30 seconds ago');
    expect(formatRelativeTime(at(-5 * MIN), now)).toBe('5 minutes ago');
    expect(formatRelativeTime(at(-3 * HOUR), now)).toBe('3 hours ago');
    expect(formatRelativeTime(at(-1 * DAY), now)).toBe('yesterday');
    expect(formatRelativeTime(at(-4 * DAY), now)).toBe('4 days ago');
  });

  it('renders future times', () => {
    expect(formatRelativeTime(at(2 * MIN), now)).toBe('in 2 minutes');
    expect(formatRelativeTime(at(1 * DAY), now)).toBe('tomorrow');
  });

  it('escalates to larger units', () => {
    expect(formatRelativeTime(at(-90 * DAY), now)).toBe('3 months ago');
    expect(formatRelativeTime(at(-800 * DAY), now)).toBe('2 years ago');
  });
});
