'use client';

import { Moon, Sun } from 'lucide-react';
import { Button } from '@shared/ui';
import { usePreferences } from './PreferencesProvider';

// One-click light/dark flip for the app bar. It sets an explicit theme (never
// 'system'), which is what a user pressing it expects; /preferences keeps the
// three-way choice including following the OS.
export default function ThemeToggle() {
  const { resolvedTheme, setTheme } = usePreferences();
  const next = resolvedTheme === 'dark' ? 'light' : 'dark';

  return (
    <Button
      variant="ghost"
      size="icon-sm"
      aria-label={`Switch to ${next} theme`}
      onClick={() => setTheme(next)}
    >
      {resolvedTheme === 'dark' ? <Sun aria-hidden /> : <Moon aria-hidden />}
    </Button>
  );
}
