'use client';

import { CalendarClock, Timer } from 'lucide-react';
import { SegmentedControl, SegmentedItem } from '@shared/ui';
import { ABSOLUTE_TIME_FORMAT, RELATIVE_TIME_FORMAT, type TimeFormat } from '@shared/preferences';
import { usePreferences } from './PreferencesProvider';

// Toggle between relative ("5 minutes ago") and absolute timestamp display.
// Writes the selection straight through to the shared preference.
export default function TimeFormatSelector() {
  const { preferences, setTimeFormat } = usePreferences();

  return (
    <SegmentedControl
      type="single"
      value={preferences.timeFormat}
      // An empty value arrives when the active item is pressed again — ignore
      // it so a format is always selected.
      onValueChange={(next: string) => next && setTimeFormat(next as TimeFormat)}
      aria-label="Timestamp format"
    >
      <SegmentedItem value={RELATIVE_TIME_FORMAT} className="flex items-center gap-1.5 px-2.5">
        <Timer className="size-3.5" aria-hidden />
        Relative
      </SegmentedItem>
      <SegmentedItem value={ABSOLUTE_TIME_FORMAT} className="flex items-center gap-1.5 px-2.5">
        <CalendarClock className="size-3.5" aria-hidden />
        Absolute
      </SegmentedItem>
    </SegmentedControl>
  );
}
