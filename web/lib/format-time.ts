// Pure timestamp formatting shared by the <TimeDisplay> component. Kept free of
// React so it can be unit-tested directly.

export interface FormattedTime {
  // Short form shown inline, e.g. "2026-06-04 01:59:00 UTC".
  text: string;
  // Long form for a tooltip / title attribute, includes the full zone name.
  title: string;
}

const SHORT_OPTIONS: Intl.DateTimeFormatOptions = {
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  hour12: false,
  timeZoneName: 'short',
};

const LONG_OPTIONS: Intl.DateTimeFormatOptions = {
  ...SHORT_OPTIONS,
  weekday: 'short',
  timeZoneName: 'long',
};

// Format an ISO-8601 timestamp for display. `timeZone` is an IANA zone name
// (e.g. 'UTC', 'Asia/Seoul'); pass undefined to use the runtime's local zone
// (the visitor's browser zone on the client). Returns null for empty/invalid
// input so callers can render their own placeholder.
export function formatTimestamp(
  value: string | null | undefined,
  timeZone?: string,
): FormattedTime | null {
  if (!value) return null;
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return null;

  // `undefined` timeZone means "host zone", stable within a session.
  const zoneKey = timeZone ?? 'local';

  return {
    text: format(date, { ...SHORT_OPTIONS, timeZone }, `short:${zoneKey}`),
    title: format(date, { ...LONG_OPTIONS, timeZone }, `long:${zoneKey}`),
  };
}

// Constructing an Intl.DateTimeFormat is the expensive part (`.format()` is
// cheap), and a single table can render hundreds of timestamps per pass — so
// cache one formatter per distinct options set keyed by timezone.
const formatterCache = new Map<string, Intl.DateTimeFormat>();

function getFormatter(options: Intl.DateTimeFormatOptions, cacheKey: string): Intl.DateTimeFormat {
  let fmt = formatterCache.get(cacheKey);
  if (!fmt) {
    fmt = new Intl.DateTimeFormat('en-CA', options);
    formatterCache.set(cacheKey, fmt);
  }
  return fmt;
}

// Intl renders "2026-06-04, 01:59:00 UTC"; drop the commas so the output reads
// as a single clean timestamp.
function format(date: Date, options: Intl.DateTimeFormatOptions, cacheKey: string): string {
  return getFormatter(options, cacheKey).format(date).replaceAll(',', '');
}

// Largest-to-smallest cascade of units and how many of the smaller unit make up
// one of this one. The walk divides the elapsed seconds by each `amount` until
// the magnitude fits inside the next unit, then formats with that unit.
const RELATIVE_DIVISIONS: { amount: number; unit: Intl.RelativeTimeFormatUnit }[] = [
  { amount: 60, unit: 'second' },
  { amount: 60, unit: 'minute' },
  { amount: 24, unit: 'hour' },
  { amount: 7, unit: 'day' },
  { amount: 4.34524, unit: 'week' },
  { amount: 12, unit: 'month' },
  { amount: Number.POSITIVE_INFINITY, unit: 'year' },
];

// `numeric: 'auto'` yields friendly phrasings ("now", "yesterday") instead of
// always-numeric ("0 seconds ago", "1 day ago").
const relativeFormatter = new Intl.RelativeTimeFormat('en', { numeric: 'auto' });

// Future timestamps within this window render as "now". Server/client clock
// skew routinely puts a just-created resource a few seconds ahead of the
// browser clock, and "in 3 seconds" reads as a glitch. Genuinely future times
// (e.g. certificate expiries) sit far beyond this and still render as "in …".
const FUTURE_SKEW_TOLERANCE_SECONDS = 10;

// Format an ISO-8601 timestamp as a relative phrase ("5 minutes ago", "in 3
// hours", "now"). `nowMs` is the reference epoch-ms — pass a shared, ticking
// value so many timestamps advance in lockstep; defaults to Date.now(). Returns
// null for empty/invalid input so callers can render their own placeholder.
export function formatRelativeTime(
  value: string | null | undefined,
  nowMs?: number,
): string | null {
  if (!value) return null;
  const time = new Date(value).getTime();
  if (Number.isNaN(time)) return null;

  // Signed seconds: negative is in the past, positive is in the future.
  let duration = (time - (nowMs ?? Date.now())) / 1000;
  if (duration > 0 && duration < FUTURE_SKEW_TOLERANCE_SECONDS) duration = 0;

  for (const { amount, unit } of RELATIVE_DIVISIONS) {
    // Round before the threshold check (not just for output): a value like
    // 59.6s would otherwise print "60 seconds ago" instead of rolling over to
    // "1 minute ago". The unrounded `duration` still feeds the next division so
    // precision is preserved across units.
    const rounded = Math.round(duration);
    if (Math.abs(rounded) < amount) {
      return relativeFormatter.format(rounded, unit);
    }
    duration /= amount;
  }
  // Unreachable: the last division has an infinite amount.
  return null;
}
