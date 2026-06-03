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
import { useApi } from '@/lib/swr';
import type { UserProfileResponse } from '@/lib/types';
import { useAuth } from './AuthProvider';

const SHOW_ALL_KEY = 'opamp.menuShowAll';

interface PermissionsContextValue {
  permissions: Set<string>;
  loading: boolean;
  hasPermission: (resource: string, action: string) => boolean;
  showAll: boolean;
  setShowAll: (v: boolean) => void;
  refresh: () => Promise<void>;
}

const PermissionsContext = createContext<PermissionsContextValue | undefined>(undefined);

// Permission strings are of the form "<resource>:<action>" (e.g. "agent:LIST").
// Roles may also hold wildcards like "agent:*", "*:LIST", or "*:*".
function matches(perms: Set<string>, resource: string, action: string): boolean {
  return (
    perms.has(`${resource}:${action}`) ||
    perms.has(`${resource}:*`) ||
    perms.has(`*:${action}`) ||
    perms.has('*:*')
  );
}

export function PermissionsProvider({ children }: { children: ReactNode }) {
  const { authenticated } = useAuth();
  const [showAll, setShowAllState] = useState(false);

  // Hydrate showAll from localStorage after mount (avoid SSR/CSR mismatch).
  // localStorage can throw in private browsing / when disabled — guard it.
  useEffect(() => {
    if (typeof window === 'undefined') return;
    try {
      const stored = window.localStorage.getItem(SHOW_ALL_KEY);
      if (stored === null) return;
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setShowAllState(stored === '1');
    } catch {
      // Keep the default (false).
    }
  }, []);

  // Shares the /api/v1/users/me request with the profile page via SWR's cache.
  // A null key while unauthenticated disables the request (no permissions).
  const {
    data: profile,
    isLoading: loading,
    mutate,
  } = useApi<UserProfileResponse>(authenticated ? '/api/v1/users/me' : null);

  const permissions = useMemo(() => {
    const set = new Set<string>();
    for (const entry of profile?.roles ?? []) {
      for (const p of entry.role.spec.permissions ?? []) {
        set.add(p);
      }
    }
    return set;
  }, [profile]);

  const refresh = useCallback(async () => {
    await mutate();
  }, [mutate]);

  const setShowAll = useCallback((v: boolean) => {
    setShowAllState(v);
    if (typeof window !== 'undefined') {
      try {
        window.localStorage.setItem(SHOW_ALL_KEY, v ? '1' : '0');
      } catch {
        // Best-effort: preference just won't persist.
      }
    }
  }, []);

  const hasPermission = useCallback(
    (resource: string, action: string) => matches(permissions, resource, action),
    [permissions],
  );

  const value = useMemo<PermissionsContextValue>(
    () => ({ permissions, loading, hasPermission, showAll, setShowAll, refresh }),
    [permissions, loading, hasPermission, showAll, setShowAll, refresh],
  );

  return <PermissionsContext.Provider value={value}>{children}</PermissionsContext.Provider>;
}

export function usePermissions(): PermissionsContextValue {
  const ctx = useContext(PermissionsContext);
  if (!ctx) throw new Error('usePermissions must be used within PermissionsProvider');
  return ctx;
}
