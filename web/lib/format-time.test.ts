import { describe, expect, it } from 'vitest';
import { formatTimestamp } from './format-time';

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
