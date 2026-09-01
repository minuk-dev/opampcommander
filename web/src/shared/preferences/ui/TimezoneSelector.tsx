'use client';

import { Globe, LocateFixed } from 'lucide-react';
import { useMemo } from 'react';
import { Button, Combobox, Field } from '@shared/ui';
import { LOCAL_TIME_ZONE, listTimeZones } from '@shared/preferences';
import { usePreferences } from './PreferencesProvider';

interface Props {
  // The visitor's resolved browser zone, shown alongside the "Local time"
  // option/button so they know what it maps to.
  localZone?: string | null;
}

// One-click presets (Local time / UTC) plus a searchable picker over every IANA
// timezone. Writes the selection straight through to the shared preference.
export default function TimezoneSelector({ localZone }: Props) {
  const { preferences, setTimeZone } = usePreferences();
  const current = preferences.timeZone;

  const options = useMemo(() => {
    const zones = [LOCAL_TIME_ZONE, ...listTimeZones()];
    // Keep a persisted-but-unlisted zone selectable so the picker can match it.
    if (!zones.includes(current)) zones.push(current);
    return zones.map((z) => ({
      value: z,
      label: z === LOCAL_TIME_ZONE ? `Local time${localZone ? ` — ${localZone}` : ''}` : z,
    }));
  }, [current, localZone]);

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap gap-1.5">
        <Button
          size="sm"
          variant={current === LOCAL_TIME_ZONE ? 'default' : 'outline'}
          onClick={() => setTimeZone(LOCAL_TIME_ZONE)}
        >
          <LocateFixed aria-hidden />
          Local time
        </Button>
        <Button
          size="sm"
          variant={current === 'UTC' ? 'default' : 'outline'}
          onClick={() => setTimeZone('UTC')}
        >
          <Globe aria-hidden />
          UTC
        </Button>
      </div>
      <Field label="Or pick a specific timezone">
        {(field) => (
          <Combobox
            {...field}
            value={current}
            onChange={setTimeZone}
            options={options}
            searchPlaceholder="Search timezones…"
            emptyMessage="No timezone matches"
          />
        )}
      </Field>
    </div>
  );
}
