'use client';

import { Clock } from 'lucide-react';
import Link from 'next/link';
import { useEffect, useState } from 'react';
import * as PopoverPrimitive from '@radix-ui/react-popover';
import { Button, Separator } from '@shared/ui';
import { LOCAL_TIME_ZONE } from '@shared/preferences';
import { usePreferences } from './PreferencesProvider';
import TimezoneSelector from './TimezoneSelector';

// Top-bar control showing the active display timezone and opening a quick
// picker. Mirrors the setting on the Preferences page (same shared state).
export default function TimezoneButton() {
  const { preferences, hydrated } = usePreferences();
  const [open, setOpen] = useState(false);
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
    <PopoverPrimitive.Root open={open} onOpenChange={setOpen}>
      <PopoverPrimitive.Trigger asChild>
        <Button variant="ghost" size="sm" className="max-w-44" aria-label="Display timezone">
          <Clock aria-hidden />
          <span className="truncate">{label}</span>
        </Button>
      </PopoverPrimitive.Trigger>
      <PopoverPrimitive.Portal>
        <PopoverPrimitive.Content
          align="end"
          sideOffset={6}
          className="z-50 w-80 rounded-lg border border-border bg-popover p-3 text-popover-foreground shadow-lg data-[state=open]:animate-in data-[state=open]:fade-in-0"
        >
          <p className="mb-2 text-xs font-medium">Display timezone</p>
          <TimezoneSelector localZone={localZone} />
          <Separator className="my-2.5" />
          <p className="text-xs text-muted-foreground">
            Applies to all timestamps in this browser. More in{' '}
            <Link href="/preferences" onClick={() => setOpen(false)} className="underline">
              Preferences
            </Link>
            .
          </p>
        </PopoverPrimitive.Content>
      </PopoverPrimitive.Portal>
    </PopoverPrimitive.Root>
  );
}
