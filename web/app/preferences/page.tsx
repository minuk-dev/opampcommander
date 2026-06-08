'use client';

import { Alert, Box, Card, CardContent, Typography } from '@mui/material';
import { useEffect, useState } from 'react';
import PageHeader from '@/components/PageHeader';
import TimeDisplay from '@/components/TimeDisplay';
import TimeFormatSelector from '@/components/TimeFormatSelector';
import TimezoneSelector from '@/components/TimezoneSelector';
import { usePreferences } from '@/components/PreferencesProvider';

export default function PreferencesPage() {
  const { hydrated } = usePreferences();

  // The visitor's resolved browser timezone, shown so they know what "Local"
  // maps to. Only meaningful client-side, so resolve it after mount.
  const [localZone, setLocalZone] = useState<string | null>(null);
  // A live "now" so the preview reflects the current time, not page-load time.
  const [now, setNow] = useState<string | null>(null);
  useEffect(() => {
    try {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setLocalZone(Intl.DateTimeFormat().resolvedOptions().timeZone);
    } catch {
      // Leave null; the label simply omits the zone name.
    }
    const tick = () => setNow(new Date().toISOString());
    tick();
    const id = setInterval(tick, 1000);
    return () => clearInterval(id);
  }, []);

  return (
    <Box>
      <PageHeader
        title="Preferences"
        subtitle="Display settings stored in this browser. They apply only to you and are never sent to the server."
      />

      <Card variant="outlined" sx={{ mb: 3, maxWidth: 640 }}>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            Time format
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            Show timestamps as a relative time (&ldquo;5 minutes ago&rdquo;) with the absolute time
            on hover, or always as the full absolute timestamp.
          </Typography>

          <TimeFormatSelector />
        </CardContent>
      </Card>

      <Card variant="outlined" sx={{ mb: 3, maxWidth: 640 }}>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            Timezone
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            How timestamps are displayed throughout the app. Choose your browser&apos;s local time,
            UTC, or any specific timezone.
          </Typography>

          <Box sx={{ maxWidth: 360 }}>
            <TimezoneSelector localZone={localZone} />
          </Box>

          <Alert severity="info" icon={false} sx={{ mt: 2 }}>
            <Typography variant="body2">
              Preview:{' '}
              <Box component="span" sx={{ fontFamily: 'var(--font-geist-mono), monospace' }}>
                {hydrated && now ? <TimeDisplay value={now} /> : '…'}
              </Box>
            </Typography>
          </Alert>
        </CardContent>
      </Card>

      <Typography variant="caption" color="text.disabled">
        More settings (theme, dark mode) will appear here.
      </Typography>
    </Box>
  );
}
