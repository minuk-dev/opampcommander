'use client';

// Shared SWR setup. SWR gives us request deduplication, in-memory caching, and
// revalidation (on focus / reconnect) across component instances, replacing the
// hand-rolled useEffect + useState fetch pattern.
//
// Keys are either a backend path string, or a [path, query] tuple so the same
// path with different query params caches separately. SWR serializes the key
// structurally, so tuples dedupe correctly.

import useSWR, { type SWRConfiguration, type SWRResponse } from 'swr';
import useSWRImmutable from 'swr/immutable';
import { api } from './api-client';

type Query = Record<string, string | number | boolean | undefined | null>;
export type SWRKey = string | readonly [string, Query];

export function fetcher<T>(key: SWRKey): Promise<T> {
  if (Array.isArray(key)) {
    const [path, query] = key as readonly [string, Query];
    return api.get<T>(path, { query });
  }
  return api.get<T>(key as string);
}

// Thin wrapper so callers don't repeat the generic fetcher. Passing `null` as
// the key disables the request (e.g. while a dependency isn't ready yet).
export function useApi<T>(key: SWRKey | null, config?: SWRConfiguration<T>): SWRResponse<T> {
  return useSWR<T>(key, fetcher, config);
}

export { useSWR, useSWRImmutable };
export type { SWRConfiguration, SWRResponse };
