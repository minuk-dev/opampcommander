// Server-side API client for React Server Components. Unlike the browser
// `api` client (which reads the token from localStorage and goes through the
// /api/proxy route), this reads the bearer token from the httpOnly session
// cookie and calls the backend directly, server-to-server.

import 'server-only';
import { cache } from 'react';
import { cookies } from 'next/headers';
import { TOKEN_COOKIE } from './cookies';

const OPAMP_API_URL = process.env.OPAMP_API_URL || 'http://localhost:8080';

// Keep in sync with NAMESPACE_COOKIE in lib/auth-storage.ts.
const NAMESPACE_COOKIE = 'opamp_namespace';
const DEFAULT_NAMESPACE = 'default';

// Namespace selection for Server Components, read from the cookie the client
// writes on selection (lib/auth-storage writeNamespace). Wrapped in React
// cache() so repeated reads across the RSC tree dedupe within a request — this
// is non-fetch work, so it isn't covered by Next's fetch memoization (guide
// 3.9).
export const readNamespace = cache(async (): Promise<string> => {
  const ns = (await cookies()).get(NAMESPACE_COOKIE)?.value;
  return ns || DEFAULT_NAMESPACE;
});

export class ServerApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.name = 'ServerApiError';
    this.status = status;
  }
}

// No React cache() wrapper needed: this is fetch-based, and Next.js already
// memoizes identical fetches within a single request (guide 3.9).
export async function serverGet<T>(path: string): Promise<T> {
  const token = (await cookies()).get(TOKEN_COOKIE)?.value;
  const clean = path.startsWith('/') ? path : `/${path}`;

  const res = await fetch(`${OPAMP_API_URL}${clean}`, {
    headers: {
      Accept: 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    // Request data is per-user and time-sensitive; never cache it.
    cache: 'no-store',
  });

  if (!res.ok) {
    throw new ServerApiError(res.status, `HTTP ${res.status} ${res.statusText}`);
  }
  return (await res.json()) as T;
}
