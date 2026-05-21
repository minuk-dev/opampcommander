// Token storage. Lives in localStorage so it survives reloads; the
// AuthProvider syncs state with the same key.

export const TOKEN_KEY = 'opamp.token';
export const REFRESH_TOKEN_KEY = 'opamp.refreshToken';
export const EXPIRES_AT_KEY = 'opamp.expiresAt';
export const NAMESPACE_KEY = 'opamp.namespace';

export interface StoredAuth {
  token: string;
  refreshToken?: string;
  expiresAt?: string;
}

export function readAuth(): StoredAuth | null {
  if (typeof window === 'undefined') return null;
  const token = window.localStorage.getItem(TOKEN_KEY);
  if (!token) return null;
  return {
    token,
    refreshToken: window.localStorage.getItem(REFRESH_TOKEN_KEY) ?? undefined,
    expiresAt: window.localStorage.getItem(EXPIRES_AT_KEY) ?? undefined,
  };
}

export function writeAuth(auth: StoredAuth): void {
  if (typeof window === 'undefined') return;
  window.localStorage.setItem(TOKEN_KEY, auth.token);
  if (auth.refreshToken) {
    window.localStorage.setItem(REFRESH_TOKEN_KEY, auth.refreshToken);
  } else {
    window.localStorage.removeItem(REFRESH_TOKEN_KEY);
  }
  if (auth.expiresAt) {
    window.localStorage.setItem(EXPIRES_AT_KEY, auth.expiresAt);
  } else {
    window.localStorage.removeItem(EXPIRES_AT_KEY);
  }
}

export function clearAuth(): void {
  if (typeof window === 'undefined') return;
  window.localStorage.removeItem(TOKEN_KEY);
  window.localStorage.removeItem(REFRESH_TOKEN_KEY);
  window.localStorage.removeItem(EXPIRES_AT_KEY);
}

export function readNamespace(): string | null {
  if (typeof window === 'undefined') return null;
  return window.localStorage.getItem(NAMESPACE_KEY);
}

export function writeNamespace(ns: string): void {
  if (typeof window === 'undefined') return;
  window.localStorage.setItem(NAMESPACE_KEY, ns);
}
