'use client';

import { AccessTime as ClockIcon } from '@mui/icons-material';
import { Box, Button, Divider, Popover, Tooltip, Typography } from '@mui/material';
import Link from 'next/link';
import { useEffect, useState } from 'react';
import { usePreferences } from './PreferencesProvider';
import TimezoneSelector from './TimezoneSelector';
import { LOCAL_TIME_ZONE } from '@/lib/preferences';

// Top-bar control showing the active display timezone and opening a quick
// picker. Mirrors the setting on the Preferences page (same shared state).
export default function TimezoneButton() {
  const { preferences, hydrated } = usePreferences();
  const [anchor, setAnchor] = useState<HTMLElement | null>(null);
  const [localZone, setLocalZone] = useState<string | null>(null);

  useEffect(() => {
    try {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setLocalZone(Intl.DateTimeFormat().resolvedOptions().timeZone);
    } catch {
      // Leave null; the label falls back to "Local".
    }
  }, []);

  // Render a stable placeholder until hydrated so the server and first client
  // render match (the resolved zone is only known in the browser).
  const label = !hydrated
    ? '…'
    : preferences.timeZone === LOCAL_TIME_ZONE
      ? (localZone ?? 'Local')
      : preferences.timeZone;

  return (
    <>
      <Tooltip title="Display timezone">
        <Button
          color="inherit"
          startIcon={<ClockIcon />}
          onClick={(e) => setAnchor(e.currentTarget)}
          sx={{ textTransform: 'none', maxWidth: 220, minWidth: 0 }}
        >
          <Box
            component="span"
            sx={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
          >
            {label}
          </Box>
        </Button>
      </Tooltip>
      <Popover
        open={Boolean(anchor)}
        anchorEl={anchor}
        onClose={() => setAnchor(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
        transformOrigin={{ vertical: 'top', horizontal: 'right' }}
        slotProps={{ paper: { sx: { p: 2, width: 340 } } }}
      >
        <Typography variant="subtitle2" gutterBottom>
          Display timezone
        </Typography>
        <TimezoneSelector localZone={localZone} autoFocus size="small" />
        <Divider sx={{ my: 1.5 }} />
        <Typography variant="caption" color="text.secondary">
          Applies to all timestamps in this browser. More in{' '}
          <Link href="/preferences" onClick={() => setAnchor(null)} style={{ color: 'inherit' }}>
            Preferences
          </Link>
          .
        </Typography>
      </Popover>
    </>
  );
}
