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
import { readNamespace, useApi, writeNamespace, type ListResponse } from '@shared/api';
import type { Namespace } from './types';
import { useAuth } from '@entities/session';

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
  // A null key keeps SWR idle until the session is authenticated. Fetch errors
  // are swallowed on purpose — a 401 is handled by the api client.
  const { data, isLoading, mutate } = useApi<ListResponse<Namespace>>(
    authenticated ? ['/api/v1/namespaces', { limit: 200 }] : null,
  );
  const namespaces = useMemo(() => data?.items ?? [], [data]);

  // If a freshly fetched list no longer contains the current selection, fall
  // back to default or the first available item. Adjusting state during render
  // (React's "derive state from props" pattern) keeps the rest of this render
  // on the namespace children will actually see.
  const [seenNamespaces, setSeenNamespaces] = useState(namespaces);
  if (namespaces !== seenNamespaces) {
    setSeenNamespaces(namespaces);
    if (namespaces.length > 0 && !namespaces.some((n) => n.metadata.name === namespace)) {
      setNamespaceState(
        namespaces.find((n) => n.metadata.name === DEFAULT_NAMESPACE)?.metadata.name ??
          namespaces[0].metadata.name,
      );
    }
  }

  // Keep the namespace cookie in step with the selection so Server Components
  // see it — on mount, and whenever the fallback above changes it.
  useEffect(() => {
    writeNamespace(namespace);
  }, [namespace]);

  const refresh = useCallback(async () => {
    await mutate();
  }, [mutate]);

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
    () => ({ namespace, setNamespace, namespaces, refresh, loading: isLoading }),
    [namespace, setNamespace, namespaces, refresh, isLoading],
  );

  return <NamespaceContext.Provider value={value}>{children}</NamespaceContext.Provider>;
}

export function useNamespace(): NamespaceContextValue {
  const ctx = useContext(NamespaceContext);
  if (!ctx) throw new Error('useNamespace must be used within NamespaceProvider');
  return ctx;
}
