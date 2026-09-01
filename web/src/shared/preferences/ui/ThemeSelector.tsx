'use client';

import { Monitor, Moon, Sun } from 'lucide-react';
import { SegmentedControl, SegmentedItem } from '@shared/ui';
import { type Theme, THEMES } from '../model/preferences';
import { usePreferences } from './PreferencesProvider';

const icons: Record<Theme, typeof Sun> = { system: Monitor, light: Sun, dark: Moon };
const labels: Record<Theme, string> = { system: 'System', light: 'Light', dark: 'Dark' };

export default function ThemeSelector() {
  const { preferences, setTheme } = usePreferences();

  return (
    <SegmentedControl
      type="single"
      value={preferences.theme}
      onValueChange={(v: string) => v && setTheme(v as Theme)}
      aria-label="Theme"
    >
      {THEMES.map((theme) => {
        const Icon = icons[theme];
        return (
          <SegmentedItem key={theme} value={theme} className="flex items-center gap-1.5 px-2.5">
            <Icon className="size-3.5" aria-hidden />
            {labels[theme]}
          </SegmentedItem>
        );
      })}
    </SegmentedControl>
  );
}
