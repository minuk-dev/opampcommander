// Client-side helpers that mirror the bearer token into an httpOnly session
// cookie via the /api/session route handler. Best-effort: a failure here only
// means Server Components can't read the token yet, not that the (localStorage)
// client session is broken.

import type { StoredAuth } from './auth-storage';

export async function setSessionCookie(auth: StoredAuth): Promise<void> {
  try {
    await fetch('/api/session', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(auth),
    });
  } catch {
    // Best-effort.
  }
}

export async function clearSessionCookie(): Promise<void> {
  try {
    await fetch('/api/session', { method: 'DELETE' });
  } catch {
    // Best-effort.
  }
}
