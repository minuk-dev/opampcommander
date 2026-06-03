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
import { useRouter } from 'next/navigation';
import { api } from '@/lib/api-client';
import { readNamespace, writeNamespace } from '@/lib/auth-storage';
import type { ListResponse, Namespace } from '@/lib/types';
import { useAuth } from './AuthProvider';

interface NamespaceContextValue {
  namespace: string;
  setNamespace: (ns: string) => void;
  namespaces: Namespace[];
  refresh: () => Promise<void>;
  loading: boolean;
}

const NamespaceContext = createContext<NamespaceContextValue | undefined>(undefined);

const DEFAULT_NAMESPACE = 'default';

export function NamespaceProvider({ children }: { children: ReactNode }) {
  const { authenticated } = useAuth();
  const router = useRouter();
  const [namespace, setNamespaceState] = useState<string>(
    () => readNamespace() ?? DEFAULT_NAMESPACE,
  );
  const [namespaces, setNamespaces] = useState<Namespace[]>([]);
  const [loading, setLoading] = useState(false);

  // Ensure the namespace cookie reflects the current selection on mount, so
  // Server Components have it even for sessions that predate cookie support.
  useEffect(() => {
    writeNamespace(namespace);
    // mount only
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const refresh = useCallback(async () => {
    if (!authenticated) return;
    setLoading(true);
    try {
      const res = await api.get<ListResponse<Namespace>>('/api/v1/namespaces', {
        query: { limit: 200 },
      });
      setNamespaces(res.items ?? []);
      // If the current selection no longer exists, fall back to default or
      // the first available item.
      const exists = (res.items ?? []).some((n) => n.metadata.name === namespace);
      if (!exists) {
        const fallback =
          (res.items ?? []).find((n) => n.metadata.name === DEFAULT_NAMESPACE)?.metadata.name ??
          res.items?.[0]?.metadata.name ??
          DEFAULT_NAMESPACE;
        setNamespaceState(fallback);
        writeNamespace(fallback);
      }
    } catch {
      // ignored — likely 401 will be handled by api-client
    } finally {
      setLoading(false);
    }
  }, [authenticated, namespace]);

  useEffect(() => {
    void refresh();
    // refresh on first authenticated mount
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [authenticated]);

  const setNamespace = useCallback(
    (ns: string) => {
      setNamespaceState(ns);
      writeNamespace(ns);
      // Re-run Server Components so RSC pages refetch for the new namespace.
      router.refresh();
    },
    [router],
  );

  const value = useMemo<NamespaceContextValue>(
    () => ({ namespace, setNamespace, namespaces, refresh, loading }),
    [namespace, setNamespace, namespaces, refresh, loading],
  );

  return <NamespaceContext.Provider value={value}>{children}</NamespaceContext.Provider>;
}

export function useNamespace(): NamespaceContextValue {
  const ctx = useContext(NamespaceContext);
  if (!ctx) throw new Error('useNamespace must be used within NamespaceProvider');
  return ctx;
}
