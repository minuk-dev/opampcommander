'use client';

import {
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from 'react';
import {
  DEFAULT_PREFERENCES,
  type Preferences,
  type Theme,
  type TimeFormat,
  readPreferences,
  resolveTheme,
  writePreferences,
} from '@shared/preferences';

interface PreferencesContextValue {
  preferences: Preferences;
  // Set the display timezone: 'local' (browser zone) or an IANA zone name.
  setTimeZone: (timeZone: string) => void;
  // Set how timestamps render: 'relative' or 'absolute'.
  setTimeFormat: (timeFormat: TimeFormat) => void;
  // Set the colour theme: 'system', 'light' or 'dark'.
  setTheme: (theme: Theme) => void;
  // The theme actually in effect, with 'system' already resolved.
  resolvedTheme: 'light' | 'dark';
  // True once the persisted preferences have been read from localStorage on the
  // client. Components that render timezone-dependent output (which differs
  // between the server and the visitor's browser) gate on this to stay
  // hydration-safe: they render a deterministic UTC value until it flips true.
  hydrated: boolean;
}

const PreferencesContext = createContext<PreferencesContextValue | undefined>(undefined);

export function PreferencesProvider({ children }: { children: ReactNode }) {
  // Start from defaults so the server render and the first client render match;
  // hydrate the real persisted values after mount.
  const [preferences, setPreferences] = useState<Preferences>(DEFAULT_PREFERENCES);
  const [hydrated, setHydrated] = useState(false);

  // Hydrate persisted preferences after mount (keeps the server render and the
  // first client render identical — see `hydrated` above).
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setPreferences(readPreferences());
    setHydrated(true);
  }, []);

  const setTimeZone = useCallback((timeZone: string) => {
    setPreferences((prev) => {
      const next = { ...prev, timeZone };
      writePreferences(next);
      return next;
    });
  }, []);

  const setTimeFormat = useCallback((timeFormat: TimeFormat) => {
    setPreferences((prev) => {
      const next = { ...prev, timeFormat };
      writePreferences(next);
      return next;
    });
  }, []);

  const setTheme = useCallback((theme: Theme) => {
    setPreferences((prev) => {
      const next = { ...prev, theme };
      writePreferences(next);
      return next;
    });
  }, []);

  // Mirror the chosen theme onto <html class="dark">, which is what the CSS
  // variables key off. The inline script in app/layout.tsx does this before
  // first paint; this keeps it in sync afterwards, including when the OS
  // setting changes while 'system' is selected.
  const [resolvedTheme, setResolvedTheme] = useState<'light' | 'dark'>('light');
  useEffect(() => {
    const apply = () => {
      const next = resolveTheme(preferences.theme);
      document.documentElement.classList.toggle('dark', next === 'dark');
      document.documentElement.style.colorScheme = next;
      setResolvedTheme(next);
    };
    apply();
    if (preferences.theme !== 'system') return;
    const media = window.matchMedia('(prefers-color-scheme: dark)');
    media.addEventListener('change', apply);
    return () => media.removeEventListener('change', apply);
  }, [preferences.theme]);

  const value = useMemo<PreferencesContextValue>(
    () => ({ preferences, setTimeZone, setTimeFormat, setTheme, resolvedTheme, hydrated }),
    [preferences, setTimeZone, setTimeFormat, setTheme, resolvedTheme, hydrated],
  );

  return <PreferencesContext.Provider value={value}>{children}</PreferencesContext.Provider>;
}

export function usePreferences(): PreferencesContextValue {
  const ctx = useContext(PreferencesContext);
  if (!ctx) throw new Error('usePreferences must be used within PreferencesProvider');
  return ctx;
}
