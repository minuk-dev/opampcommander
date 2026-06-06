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
