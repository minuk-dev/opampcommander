// Single client-side fetch wrapper. Calls the Next.js catch-all proxy at
// /api/proxy/<backend-path>, attaches the bearer token from localStorage,
// and surfaces backend error payloads (RFC 9457 problem details).

import { readAuth, writeAuth, clearAuth } from './auth-storage';

export interface ApiError extends Error {
  status: number;
  body?: unknown;
}

function buildUrl(path: string): string {
  // path is the backend path, e.g. "/api/v1/agents".
  const clean = path.startsWith('/') ? path : `/${path}`;
  return `/api/proxy${clean}`;
}

async function readBody(res: Response): Promise<unknown> {
  const ct = res.headers.get('content-type') || '';
  if (ct.includes('application/json') || ct.includes('+json')) {
    try {
      return await res.json();
    } catch {
      return undefined;
    }
  }
  try {
    const text = await res.text();
    return text || undefined;
  } catch {
    return undefined;
  }
}

async function refreshToken(): Promise<boolean> {
  const auth = readAuth();
  if (!auth?.refreshToken) return false;
  try {
    const res = await fetch(buildUrl('/api/v1/auth/refresh'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refreshToken: auth.refreshToken }),
    });
    if (!res.ok) return false;
    const data = (await res.json()) as {
      token: string;
      refreshToken?: string;
      expiresAt?: string;
    };
    writeAuth({
      token: data.token,
      refreshToken: data.refreshToken,
      expiresAt: data.expiresAt,
    });
    return true;
  } catch {
    return false;
  }
}

export interface RequestOptions {
  method?: string;
  query?: Record<string, string | number | boolean | undefined | null>;
  body?: unknown;
  headers?: Record<string, string>;
  // When true, do not redirect to /login on a 401. Used for the login page
  // itself and the /me probe.
  noAuthRedirect?: boolean;
  // If set, send Basic auth instead of Bearer. Used for the basic login call.
  basicAuth?: { username: string; password: string };
  signal?: AbortSignal;
}

async function doFetch<T>(
  path: string,
  opts: RequestOptions,
): Promise<T> {
  const url = new URL(buildUrl(path), window.location.origin);
  if (opts.query) {
    for (const [k, v] of Object.entries(opts.query)) {
      if (v === undefined || v === null || v === '') continue;
      url.searchParams.set(k, String(v));
    }
  }

  const headers: Record<string, string> = {
    Accept: 'application/json',
    ...(opts.headers ?? {}),
  };

  if (opts.basicAuth) {
    headers.Authorization =
      'Basic ' +
      btoa(`${opts.basicAuth.username}:${opts.basicAuth.password}`);
  } else {
    const auth = readAuth();
    if (auth?.token) {
      headers.Authorization = `Bearer ${auth.token}`;
    }
  }

  const init: RequestInit = {
    method: opts.method ?? 'GET',
    headers,
    signal: opts.signal,
  };

  if (opts.body !== undefined) {
    if (!headers['Content-Type']) {
      headers['Content-Type'] = 'application/json';
    }
    init.body =
      typeof opts.body === 'string' ? opts.body : JSON.stringify(opts.body);
  }

  let res = await fetch(url.toString(), init);

  if (res.status === 401 && !opts.basicAuth && !opts.noAuthRedirect) {
    const refreshed = await refreshToken();
    if (refreshed) {
      const auth = readAuth();
      if (auth?.token) headers.Authorization = `Bearer ${auth.token}`;
      res = await fetch(url.toString(), init);
    }
    if (res.status === 401) {
      clearAuth();
      const here = window.location.pathname + window.location.search;
      if (window.location.pathname !== '/login') {
        window.location.assign(
          `/login?from=${encodeURIComponent(here)}`,
        );
      }
      const err: ApiError = Object.assign(new Error('Unauthorized'), {
        status: 401,
      });
      throw err;
    }
  }

  if (!res.ok) {
    const body = await readBody(res);
    const message =
      (body && typeof body === 'object' && 'detail' in body
        ? String((body as { detail?: unknown }).detail)
        : undefined) ||
      (body && typeof body === 'object' && 'title' in body
        ? String((body as { title?: unknown }).title)
        : undefined) ||
      (body && typeof body === 'object' && 'error' in body
        ? String((body as { error?: unknown }).error)
        : undefined) ||
      `HTTP ${res.status} ${res.statusText}`;
    const err: ApiError = Object.assign(new Error(message), {
      status: res.status,
      body,
    });
    throw err;
  }

  if (res.status === 204) return undefined as T;
  const ct = res.headers.get('content-type') || '';
  if (ct.includes('application/json') || ct.includes('+json')) {
    return (await res.json()) as T;
  }
  // Fallback: parse as text.
  return (await res.text()) as unknown as T;
}

export const api = {
  get: <T>(path: string, opts: Omit<RequestOptions, 'method' | 'body'> = {}) =>
    doFetch<T>(path, { ...opts, method: 'GET' }),
  post: <T>(path: string, body?: unknown, opts: Omit<RequestOptions, 'method' | 'body'> = {}) =>
    doFetch<T>(path, { ...opts, method: 'POST', body }),
  put: <T>(path: string, body?: unknown, opts: Omit<RequestOptions, 'method' | 'body'> = {}) =>
    doFetch<T>(path, { ...opts, method: 'PUT', body }),
  patch: <T>(path: string, body?: unknown, opts: Omit<RequestOptions, 'method' | 'body'> = {}) =>
    doFetch<T>(path, { ...opts, method: 'PATCH', body }),
  delete: <T = void>(path: string, opts: Omit<RequestOptions, 'method' | 'body'> = {}) =>
    doFetch<T>(path, { ...opts, method: 'DELETE' }),
};

export type ApiClient = typeof api;
