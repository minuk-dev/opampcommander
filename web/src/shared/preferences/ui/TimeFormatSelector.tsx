'use client';

import { ToggleButton, ToggleButtonGroup } from '@mui/material';
import { Schedule as RelativeIcon, Event as AbsoluteIcon } from '@mui/icons-material';
import { usePreferences } from './PreferencesProvider';
import { ABSOLUTE_TIME_FORMAT, RELATIVE_TIME_FORMAT, type TimeFormat } from '@shared/preferences';

// Toggle between relative ("5 minutes ago") and absolute timestamp display.
// Writes the selection straight through to the shared preference.
export default function TimeFormatSelector() {
  const { preferences, setTimeFormat } = usePreferences();

  return (
    <ToggleButtonGroup
      exclusive
      color="primary"
      size="small"
      value={preferences.timeFormat}
      onChange={(_, next: TimeFormat | null) => {
        // null arrives when the active button is clicked again — ignore it so a
        // format is always selected.
        if (next) setTimeFormat(next);
      }}
      aria-label="Timestamp format"
    >
      <ToggleButton value={RELATIVE_TIME_FORMAT}>
        <RelativeIcon fontSize="small" sx={{ mr: 1 }} />
        Relative
      </ToggleButton>
      <ToggleButton value={ABSOLUTE_TIME_FORMAT}>
        <AbsoluteIcon fontSize="small" sx={{ mr: 1 }} />
        Absolute
      </ToggleButton>
    </ToggleButtonGroup>
  );
}
