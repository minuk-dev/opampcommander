'use client';

// Per-table column visibility, persisted to localStorage so each user's choice
// of which columns to show survives reloads. Purely presentational UI state
// (like the timezone preference in `preferences.ts`), so localStorage — keyed
// per table — is the right home; nothing is ever sent to the server.

import { useCallback, useEffect, useMemo, useState } from 'react';

export interface ColumnConfig {
  // Stable identifier used as the persistence key and React key. Renaming an id
  // resets that column to its default for existing users.
  id: string;
  // Label shown in the column picker menu.
  label: string;
  // Shown when the user has no saved preference. Defaults to true.
  defaultVisible?: boolean;
  // Locked columns are always visible and cannot be toggled off (e.g. the
  // primary identifier column). They still appear in the picker, checked and
  // disabled, so the full column set is discoverable.
  locked?: boolean;
}

export type ColumnVisibility = Record<string, boolean>;

const KEY_PREFIX = 'opamp.columns.';

// Persisted format version. Treat anything that doesn't match the current
// schema (older/newer versions, hand-edited junk, an unrelated value that
// happens to share the key) as incompatible and fall back to defaults. Bump
// this whenever the stored shape changes incompatibly.
const SCHEMA_VERSION = 1;

// The on-disk shape. Kept deliberately small and explicit so we can validate it
// field-by-field on read — localStorage is shared, user-writable, and may carry
// values written by an older or newer build, so we never trust its contents.
interface StoredVisibility {
  v: number;
  columns: Record<string, boolean>;
}

function storageKey(table: string): string {
  return `${KEY_PREFIX}${table}`;
}

function computeDefaults(columns: ColumnConfig[]): ColumnVisibility {
  const out: ColumnVisibility = {};
  for (const c of columns) {
    out[c.id] = c.locked ? true : (c.defaultVisible ?? true);
  }
  return out;
}

// Validate an untrusted parsed value against the current schema. Returns the
// stored column map only when the whole envelope is well-formed and the version
// matches; otherwise null so the caller can fall back to defaults.
function parseStored(value: unknown): Record<string, boolean> | null {
  if (typeof value !== 'object' || value === null) return null;
  const obj = value as Record<string, unknown>;
  if (obj.v !== SCHEMA_VERSION) return null;
  const cols = obj.columns;
  if (typeof cols !== 'object' || cols === null) return null;
  const out: Record<string, boolean> = {};
  for (const [k, v] of Object.entries(cols)) {
    // Drop any non-boolean entry rather than rejecting the whole blob; a single
    // bad field shouldn't wipe the user's other choices.
    if (typeof v === 'boolean') out[k] = v;
  }
  return out;
}

// Merge persisted overrides onto the defaults. Unknown / removed column ids in
// storage are ignored, and columns added since the preference was saved fall
// back to their default — so the table keeps working across schema changes.
// Any corruption or version mismatch degrades silently to defaults.
function readVisibility(
  table: string,
  columns: ColumnConfig[],
  defaults: ColumnVisibility,
): ColumnVisibility {
  if (typeof window === 'undefined') return defaults;
  try {
    const raw = window.localStorage.getItem(storageKey(table));
    if (!raw) return defaults;
    const stored = parseStored(JSON.parse(raw));
    if (!stored) return defaults;
    const out = { ...defaults };
    for (const c of columns) {
      if (c.locked) continue;
      if (typeof stored[c.id] === 'boolean') out[c.id] = stored[c.id];
    }
    return out;
  } catch {
    return defaults;
  }
}

function writeVisibility(table: string, visible: ColumnVisibility): void {
  if (typeof window === 'undefined') return;
  try {
    const payload: StoredVisibility = { v: SCHEMA_VERSION, columns: visible };
    window.localStorage.setItem(storageKey(table), JSON.stringify(payload));
  } catch {
    // Best-effort: the choice just won't persist across reloads.
  }
}

export interface UseColumnVisibility {
  visible: ColumnVisibility;
  isVisible: (id: string) => boolean;
  toggle: (id: string) => void;
}

// Pass a module-level `columns` array (stable identity) so the persisted values
// are read once after mount. Starting from defaults keeps the server render and
// first client render identical (hydration-safe); the saved values apply after
// mount, matching the timezone-preference pattern.
export function useColumnVisibility(table: string, columns: ColumnConfig[]): UseColumnVisibility {
  const defaults = useMemo(() => computeDefaults(columns), [columns]);
  const [visible, setVisible] = useState<ColumnVisibility>(defaults);

  useEffect(() => {
    setVisible(readVisibility(table, columns, defaults));
  }, [table, columns, defaults]);

  const toggle = useCallback(
    (id: string) => {
      setVisible((prev) => {
        const next = { ...prev, [id]: !prev[id] };
        writeVisibility(table, next);
        return next;
      });
    },
    [table],
  );

  const isVisible = useCallback((id: string) => visible[id] ?? true, [visible]);

  return { visible, isVisible, toggle };
}
