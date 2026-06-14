'use client';

import { usePreferences } from './PreferencesProvider';
import { formatRelativeTime, formatTimestamp } from '@shared/lib';
import { LOCAL_TIME_ZONE, RELATIVE_TIME_FORMAT, useNow } from '@shared/preferences';

interface Props {
  // ISO-8601 timestamp. Empty/undefined renders `emptyText`.
  value: string | null | undefined;
  // Placeholder when there is no value. Defaults to a hyphen.
  emptyText?: string;
}

// Renders a timestamp per the user's preferences. By default it shows a live
// relative time ("5 minutes ago") with the absolute timestamp in the hover
// tooltip; the 'absolute' time-format preference shows the full timestamp
// inline instead. Both honour the chosen timezone.
//
// Until preferences have hydrated on the client we render the absolute time in
// UTC, which is identical on the server and the browser — so switching to the
// relative phrasing / chosen zone happens after hydration without a mismatch.
// suppressHydrationWarning guards against any residual Intl differences between
// the Node and browser runtimes.
export default function TimeDisplay({ value, emptyText = '-' }: Props) {
  const { preferences, hydrated } = usePreferences();

  if (!value) return <>{emptyText}</>;

  // Before hydration: deterministic UTC. After: 'local' → host zone (undefined),
  // otherwise the chosen IANA zone.
  const timeZone = !hydrated
    ? 'UTC'
    : preferences.timeZone === LOCAL_TIME_ZONE
      ? undefined
      : preferences.timeZone;
  const formatted = formatTimestamp(value, timeZone);
  if (!formatted) {
    // Not a parseable timestamp — fall back to the raw value verbatim.
    return <span title={value}>{value}</span>;
  }

  // Relative phrasing depends on the client clock, so only after hydration and
  // only when the user hasn't opted into absolute display. The tooltip always
  // carries the full absolute timestamp.
  const showRelative = hydrated && preferences.timeFormat === RELATIVE_TIME_FORMAT;

  return (
    <time dateTime={value} title={formatted.title} suppressHydrationWarning>
      {showRelative ? <RelativeText value={value} fallback={formatted.text} /> : formatted.text}
    </time>
  );
}

// The live relative phrase, isolated so the 30s ticker subscription exists only
// when relative display is actually shown — in absolute mode (or pre-hydration)
// no TimeDisplay subscribes, so nothing re-renders on tick.
function RelativeText({ value, fallback }: { value: string; fallback: string }) {
  const now = useNow();
  return <>{formatRelativeTime(value, now) ?? fallback}</>;
}
