'use client';

import { useEffect, useState } from 'react';
import {
  ThemeSelector,
  TimeDisplay,
  TimeFormatSelector,
  TimezoneSelector,
  usePreferences,
} from '@shared/preferences';
import { Card, CardContent, CardDescription, CardHeader, CardTitle, PageHeader } from '@shared/ui';

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
    <div>
      <PageHeader
        title="Preferences"
        subtitle="Display settings stored in this browser. They apply only to you and are never sent to the server."
      />

      <div className="max-w-2xl space-y-3">
        <Card>
          <CardHeader>
            <CardTitle>Theme</CardTitle>
            <CardDescription>
              Follow your operating system, or pin the interface to light or dark.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <ThemeSelector />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Time format</CardTitle>
            <CardDescription>
              Show timestamps as a relative time (&ldquo;5 minutes ago&rdquo;) with the absolute
              time on hover, or always as the full absolute timestamp.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <TimeFormatSelector />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Timezone</CardTitle>
            <CardDescription>
              How timestamps are displayed throughout the app. Choose your browser&apos;s local
              time, UTC, or any specific timezone.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="max-w-sm">
              <TimezoneSelector localZone={localZone} />
            </div>
            <p className="text-sm text-muted-foreground">
              Preview:{' '}
              <span className="font-mono text-foreground">
                {hydrated && now ? <TimeDisplay value={now} /> : '…'}
              </span>
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
