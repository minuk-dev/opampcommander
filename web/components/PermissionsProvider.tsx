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
import { api } from '@/lib/api-client';
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
  const [permissions, setPermissions] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(true);
  const [showAll, setShowAllState] = useState(false);

  // Hydrate showAll from localStorage after mount (avoid SSR/CSR mismatch).
  useEffect(() => {
    if (typeof window === 'undefined') return;
    const stored = window.localStorage.getItem(SHOW_ALL_KEY);
    if (stored === null) return;
    setShowAllState(stored === '1');
  }, []);

  const refresh = useCallback(async () => {
    if (!authenticated) {
      setPermissions(new Set());
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const profile = await api.get<UserProfileResponse>('/api/v1/users/me');
      const set = new Set<string>();
      for (const entry of profile.roles ?? []) {
        for (const p of entry.role.spec.permissions ?? []) {
          set.add(p);
        }
      }
      setPermissions(set);
    } catch {
      // On failure leave permissions empty; the toggle still lets the user
      // navigate. api-client handles 401 globally.
      setPermissions(new Set());
    } finally {
      setLoading(false);
    }
  }, [authenticated]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const setShowAll = useCallback((v: boolean) => {
    setShowAllState(v);
    if (typeof window !== 'undefined') {
      window.localStorage.setItem(SHOW_ALL_KEY, v ? '1' : '0');
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
