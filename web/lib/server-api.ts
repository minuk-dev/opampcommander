// Server-side API client for React Server Components. Unlike the browser
// `api` client (which reads the token from localStorage and goes through the
// /api/proxy route), this reads the bearer token from the httpOnly session
// cookie and calls the backend directly, server-to-server.

import 'server-only';
import { cookies } from 'next/headers';
import { TOKEN_COOKIE } from '@/app/api/session/route';

const OPAMP_API_URL = process.env.OPAMP_API_URL || 'http://localhost:8080';

export class ServerApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.name = 'ServerApiError';
    this.status = status;
  }
}

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
