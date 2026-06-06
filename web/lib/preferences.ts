// Client-side, frontend-only user preferences (timezone display, and later
// theme / dark mode and other UI toggles). Persisted to localStorage so the
// choice survives reloads. These are purely presentational and never sent to
// the server, so localStorage (not a cookie/session) is the right home.

// The sentinel for "use the visitor's browser timezone". Any other value is an
// IANA timezone name (e.g. 'UTC', 'Asia/Seoul', 'America/New_York').
export const LOCAL_TIME_ZONE = 'local';

export interface Preferences {
  // 'local' (browser zone) or an IANA timezone name.
  timeZone: string;
}

export const DEFAULT_PREFERENCES: Preferences = {
  timeZone: LOCAL_TIME_ZONE,
};

const STORAGE_KEY = 'opamp.preferences';

// localStorage access throws in private browsing, when storage is disabled, or
// over quota. Preferences are best-effort UI state, so every access degrades
// to the defaults rather than throwing.
export function readPreferences(): Preferences {
  if (typeof window === 'undefined') return DEFAULT_PREFERENCES;
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return DEFAULT_PREFERENCES;
    const parsed = JSON.parse(raw) as Partial<Preferences>;
    return normalize(parsed);
  } catch {
    return DEFAULT_PREFERENCES;
  }
}

export function writePreferences(prefs: Preferences): void {
  if (typeof window === 'undefined') return;
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(prefs));
  } catch {
    // Best-effort: the preference just won't persist across reloads.
  }
}

// True for 'local' and any IANA zone the runtime recognises. Used both to
// sanitise persisted values and to validate user selections.
export function isValidTimeZone(tz: unknown): tz is string {
  if (typeof tz !== 'string') return false;
  if (tz === LOCAL_TIME_ZONE) return true;
  try {
    new Intl.DateTimeFormat('en-CA', { timeZone: tz });
    return true;
  } catch {
    return false;
  }
}

// Return the canonical IANA spelling of a zone (Intl is case-insensitive, so
// 'utc' resolves to 'UTC'). Unknown values collapse to the local sentinel.
// This also migrates the legacy lowercase 'utc' preference to 'UTC'.
export function canonicalTimeZone(tz: unknown): string {
  if (typeof tz !== 'string' || tz === LOCAL_TIME_ZONE) return LOCAL_TIME_ZONE;
  try {
    return new Intl.DateTimeFormat('en-CA', { timeZone: tz }).resolvedOptions().timeZone;
  } catch {
    return LOCAL_TIME_ZONE;
  }
}

// The list of selectable IANA timezones. Modern browsers/Node expose the full
// IANA set via Intl.supportedValuesOf; fall back to a small curated list when
// it is unavailable so the picker is never empty.
export function listTimeZones(): string[] {
  let zones = FALLBACK_ZONES;
  try {
    const supported = (Intl as typeof Intl & { supportedValuesOf?: (key: string) => string[] })
      .supportedValuesOf;
    if (typeof supported === 'function') {
      zones = supported('timeZone');
    }
  } catch {
    // Keep the curated list.
  }
  // Intl.supportedValuesOf omits 'UTC'; surface it at the top so it's always
  // an easy choice.
  return zones.includes('UTC') ? zones : ['UTC', ...zones];
}

const FALLBACK_ZONES = [
  'UTC',
  'America/Los_Angeles',
  'America/New_York',
  'America/Sao_Paulo',
  'Europe/London',
  'Europe/Berlin',
  'Europe/Moscow',
  'Asia/Dubai',
  'Asia/Kolkata',
  'Asia/Shanghai',
  'Asia/Seoul',
  'Asia/Tokyo',
  'Australia/Sydney',
];

// Coerce an untrusted parsed object into a valid Preferences, canonicalising
// the zone spelling and dropping unknown values back to their defaults.
function normalize(parsed: Partial<Preferences>): Preferences {
  return { timeZone: canonicalTimeZone(parsed.timeZone) };
}
