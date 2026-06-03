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
import { usePathname, useRouter } from 'next/navigation';
import { CircularProgress, Box } from '@mui/material';
import { api, type ApiError } from '@/lib/api-client';
import { type StoredAuth, clearAuth, readAuth, writeAuth } from '@/lib/auth-storage';
import { clearSessionCookie, setSessionCookie } from '@/lib/session';
import type { AuthInfo } from '@/lib/types';

interface AuthContextValue {
  authenticated: boolean;
  email: string | null;
  token: string | null;
  loading: boolean;
  loginBasic: (username: string, password: string) => Promise<void>;
  applyTokens: (a: StoredAuth) => Promise<void>;
  logout: () => void;
  refresh: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

const PUBLIC_ROUTES = new Set<string>(['/login', '/login/github/callback']);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setToken] = useState<string | null>(null);
  const [email, setEmail] = useState<string | null>(null);
  const [authenticated, setAuthenticated] = useState(false);
  const [loading, setLoading] = useState(true);
  const router = useRouter();
  const pathname = usePathname();

  const refresh = useCallback(async () => {
    const auth = readAuth();
    if (!auth?.token) {
      setToken(null);
      setEmail(null);
      setAuthenticated(false);
      return;
    }
    setToken(auth.token);
    // Migrate sessions that predate cookie auth: mirror the localStorage token
    // into the httpOnly cookie so Server Components can read it.
    void setSessionCookie(auth);
    try {
      const info = await api.get<AuthInfo>('/api/v1/auth/info', {
        noAuthRedirect: true,
      });
      setAuthenticated(info.authenticated);
      setEmail(info.email ?? null);
      if (!info.authenticated) {
        clearAuth();
        setToken(null);
      }
    } catch (err) {
      // Only wipe the session for an explicit 401. Transient network or
      // server errors should leave the (possibly still valid) token in
      // place so we don't log the user out on a flaky connection.
      const status = (err as ApiError)?.status;
      if (status === 401) {
        clearAuth();
        setToken(null);
        setAuthenticated(false);
        setEmail(null);
      }
    }
  }, []);

  useEffect(() => {
    void (async () => {
      await refresh();
      setLoading(false);
    })();
  }, [refresh]);

  // Redirect unauthenticated users to /login (and back when they sign in).
  useEffect(() => {
    if (loading) return;
    if (authenticated) return;
    if (PUBLIC_ROUTES.has(pathname)) return;
    const qs = typeof window !== 'undefined' ? window.location.search : '';
    const from = encodeURIComponent(pathname + (qs || ''));
    router.replace(`/login?from=${from}`);
  }, [loading, authenticated, pathname, router]);

  const loginBasic = useCallback(
    async (username: string, password: string) => {
      const data = await api.get<{
        token: string;
        refreshToken?: string;
        expiresAt?: string;
      }>('/api/v1/auth/basic', {
        basicAuth: { username, password },
        noAuthRedirect: true,
      });
      const stored: StoredAuth = {
        token: data.token,
        refreshToken: data.refreshToken,
        expiresAt: data.expiresAt,
      };
      writeAuth(stored);
      await setSessionCookie(stored);
      await refresh();
    },
    [refresh],
  );

  const applyTokens = useCallback(
    async (a: StoredAuth) => {
      writeAuth(a);
      // Await the cookie so middleware sees the session on the next navigation.
      await setSessionCookie(a);
      await refresh();
    },
    [refresh],
  );

  const logout = useCallback(() => {
    clearAuth();
    void clearSessionCookie();
    setToken(null);
    setEmail(null);
    setAuthenticated(false);
    router.replace('/login');
  }, [router]);

  const value = useMemo<AuthContextValue>(
    () => ({
      authenticated,
      email,
      token,
      loading,
      loginBasic,
      applyTokens,
      logout,
      refresh,
    }),
    [authenticated, email, token, loading, loginBasic, applyTokens, logout, refresh],
  );

  if (loading) {
    return (
      <Box display="flex" alignItems="center" justifyContent="center" minHeight="100vh">
        <CircularProgress />
      </Box>
    );
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
