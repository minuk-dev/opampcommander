'use client';

import { usePreferences } from './PreferencesProvider';
import { formatTimestamp } from '@/lib/format-time';
import { LOCAL_TIME_ZONE } from '@/lib/preferences';

interface Props {
  // ISO-8601 timestamp. Empty/undefined renders `emptyText`.
  value: string | null | undefined;
  // Placeholder when there is no value. Defaults to a hyphen.
  emptyText?: string;
}

// Renders a timestamp formatted per the user's timezone preference. Until the
// preferences have hydrated on the client we render in UTC, which is identical
// on the server and the browser — so switching to the chosen zone happens after
// hydration without a mismatch. suppressHydrationWarning guards against any
// residual Intl differences between the Node and browser runtimes.
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

  return (
    <time dateTime={value} title={formatted.title} suppressHydrationWarning>
      {formatted.text}
    </time>
  );
}
