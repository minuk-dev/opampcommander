'use client';

import { Autocomplete, Button, Stack, TextField } from '@mui/material';
import { MyLocation as LocalIcon, Public as UtcIcon } from '@mui/icons-material';
import { useMemo } from 'react';
import { usePreferences } from './PreferencesProvider';
import { LOCAL_TIME_ZONE, listTimeZones } from '@/lib/preferences';

interface Props {
  // The visitor's resolved browser zone, shown alongside the "Local time"
  // option/button so they know what it maps to.
  localZone?: string | null;
  autoFocus?: boolean;
  size?: 'small' | 'medium';
}

// One-click presets (Local time / UTC) plus a searchable picker over every IANA
// timezone. Writes the selection straight through to the shared preference.
export default function TimezoneSelector({ localZone, autoFocus, size = 'medium' }: Props) {
  const { preferences, setTimeZone } = usePreferences();
  const current = preferences.timeZone;

  const options = useMemo(() => {
    const zones = [LOCAL_TIME_ZONE, ...listTimeZones()];
    // Keep a persisted-but-unlisted zone selectable so Autocomplete can match it.
    if (!zones.includes(current)) zones.push(current);
    return zones;
  }, [current]);

  return (
    <Stack spacing={1.5}>
      <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
        <Button
          size="small"
          startIcon={<LocalIcon />}
          variant={current === LOCAL_TIME_ZONE ? 'contained' : 'outlined'}
          onClick={() => setTimeZone(LOCAL_TIME_ZONE)}
        >
          Local time
        </Button>
        <Button
          size="small"
          startIcon={<UtcIcon />}
          variant={current === 'UTC' ? 'contained' : 'outlined'}
          onClick={() => setTimeZone('UTC')}
        >
          UTC
        </Button>
      </Stack>
      <Autocomplete
        options={options}
        value={current}
        onChange={(_, v) => v && setTimeZone(v)}
        getOptionLabel={(z) =>
          z === LOCAL_TIME_ZONE ? `Local time${localZone ? ` — ${localZone}` : ''}` : z
        }
        disableClearable
        autoHighlight
        fullWidth
        size={size}
        renderInput={(params) => (
          <TextField {...params} label="Or pick a specific timezone" autoFocus={autoFocus} />
        )}
      />
    </Stack>
  );
}
