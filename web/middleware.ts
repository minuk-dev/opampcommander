import { type NextRequest, NextResponse } from 'next/server';

// Server-side auth gate. Protected app routes require the httpOnly session
// cookie set after login (see app/api/session). Requests without it are
// redirected to /login with a `from` so we can return there after sign-in.
//
// This only checks cookie *presence*, not validity — an expired token still
// reaches the page, whose data fetch 401s and triggers the client refresh /
// login bounce. Users mid-migration (localStorage token but no cookie yet)
// self-heal: they bounce to /login, where AuthProvider mirrors the token into
// the cookie and the login page sends them straight back.

const TOKEN_COOKIE = 'opamp_token';

export function middleware(req: NextRequest): NextResponse {
  const hasSession = req.cookies.has(TOKEN_COOKIE);
  if (hasSession) {
    return NextResponse.next();
  }

  const url = req.nextUrl.clone();
  const from = req.nextUrl.pathname + req.nextUrl.search;
  url.pathname = '/login';
  url.search = `?from=${encodeURIComponent(from)}`;
  return NextResponse.redirect(url);
}

export const config = {
  // Run on everything except: API routes (proxy/session handle their own
  // auth), Next internals, the login pages, and static files (those with a
  // dot in the last segment, e.g. favicon.ico).
  matcher: ['/((?!api|_next/static|_next/image|favicon.ico|login|.*\\..*).*)'],
};
