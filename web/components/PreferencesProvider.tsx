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
  type TimeFormat,
  readPreferences,
  writePreferences,
} from '@/lib/preferences';

interface PreferencesContextValue {
  preferences: Preferences;
  // Set the display timezone: 'local' (browser zone) or an IANA zone name.
  setTimeZone: (timeZone: string) => void;
  // Set how timestamps render: 'relative' or 'absolute'.
  setTimeFormat: (timeFormat: TimeFormat) => void;
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

  const value = useMemo<PreferencesContextValue>(
    () => ({ preferences, setTimeZone, setTimeFormat, hydrated }),
    [preferences, setTimeZone, setTimeFormat, hydrated],
  );

  return <PreferencesContext.Provider value={value}>{children}</PreferencesContext.Provider>;
}

export function usePreferences(): PreferencesContextValue {
  const ctx = useContext(PreferencesContext);
  if (!ctx) throw new Error('usePreferences must be used within PreferencesProvider');
  return ctx;
}
