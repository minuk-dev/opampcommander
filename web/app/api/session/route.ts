// Session cookie endpoint. The client obtains a bearer token (basic login or
// GitHub OAuth callback) and POSTs it here so we can store it in an httpOnly
// cookie. Server Components / middleware / the proxy can then read the token
// server-side — which JavaScript-readable localStorage cannot provide. httpOnly
// also keeps the token out of reach of XSS.
//
// This runs alongside the existing localStorage flow (additive); the client
// keeps its own copy for client-side fetches.

import { type NextRequest, NextResponse } from 'next/server';
import { cookies } from 'next/headers';

export const TOKEN_COOKIE = 'opamp_token';
export const REFRESH_COOKIE = 'opamp_refresh_token';
export const EXPIRES_COOKIE = 'opamp_expires_at';

interface SessionBody {
  token?: string;
  refreshToken?: string;
  expiresAt?: string;
}

function isSecure(req: NextRequest): boolean {
  return req.headers.get('x-forwarded-proto') === 'https' || req.nextUrl.protocol === 'https:';
}

export async function POST(req: NextRequest): Promise<NextResponse> {
  let body: SessionBody;
  try {
    body = (await req.json()) as SessionBody;
  } catch {
    return NextResponse.json({ error: 'invalid_body' }, { status: 400 });
  }
  if (!body.token) {
    return NextResponse.json({ error: 'token_required' }, { status: 400 });
  }

  const jar = await cookies();
  const base = {
    httpOnly: true,
    sameSite: 'lax' as const,
    secure: isSecure(req),
    path: '/',
  };

  jar.set(TOKEN_COOKIE, body.token, base);
  if (body.refreshToken) {
    jar.set(REFRESH_COOKIE, body.refreshToken, base);
  } else {
    jar.delete(REFRESH_COOKIE);
  }
  if (body.expiresAt) {
    jar.set(EXPIRES_COOKIE, body.expiresAt, base);
  } else {
    jar.delete(EXPIRES_COOKIE);
  }

  return NextResponse.json({ ok: true });
}

export async function DELETE(): Promise<NextResponse> {
  const jar = await cookies();
  jar.delete(TOKEN_COOKIE);
  jar.delete(REFRESH_COOKIE);
  jar.delete(EXPIRES_COOKIE);
  return NextResponse.json({ ok: true });
}
