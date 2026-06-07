import { type NextRequest, NextResponse } from 'next/server';

const OPAMP_API_URL = process.env.OPAMP_API_URL || 'http://localhost:8080';

const HOP_BY_HOP = new Set([
  'connection',
  'keep-alive',
  'proxy-authenticate',
  'proxy-authorization',
  'te',
  'trailer',
  'transfer-encoding',
  'upgrade',
  'content-length',
  'host',
]);

async function forward(request: NextRequest, segments: string[]): Promise<NextResponse> {
  const targetPath = `/${segments.map((s) => encodeURIComponent(s)).join('/')}`;
  const search = request.nextUrl.search;
  const target = `${OPAMP_API_URL}${targetPath}${search}`;

  const headers = new Headers();
  request.headers.forEach((value, key) => {
    if (!HOP_BY_HOP.has(key.toLowerCase())) {
      headers.set(key, value);
    }
  });
  headers.delete('host');

  // Fall back to the httpOnly session cookie when the request didn't carry a
  // bearer header (e.g. a server-side / RSC fetch that only forwards cookies).
  if (!headers.has('authorization')) {
    const token = request.cookies.get('opamp_token')?.value;
    if (token) {
      headers.set('authorization', `Bearer ${token}`);
    }
  }

  const init: RequestInit = {
    method: request.method,
    headers,
    redirect: 'manual',
  };

  if (request.method !== 'GET' && request.method !== 'HEAD') {
    init.body = await request.arrayBuffer();
  }

  let upstream: Response;
  try {
    upstream = await fetch(target, init);
  } catch (err) {
    // Log full details server-side; do not leak the upstream URL to the
    // browser (it may differ from what the user knows about, and at minimum
    // exposes internal hostnames).
    console.error('proxy upstream unreachable', { target, err });
    return NextResponse.json(
      {
        error: 'upstream_unreachable',
        message: err instanceof Error ? err.message : 'unknown',
      },
      { status: 502 },
    );
  }

  const responseHeaders = new Headers();
  upstream.headers.forEach((value, key) => {
    if (!HOP_BY_HOP.has(key.toLowerCase())) {
      responseHeaders.set(key, value);
    }
  });

  // If backend issued a redirect (e.g. /auth/github → GitHub), surface the
  // Location so the browser can follow it.
  if (upstream.status >= 300 && upstream.status < 400) {
    const location = upstream.headers.get('location');
    if (location) {
      return NextResponse.redirect(location, upstream.status as 301 | 302 | 307 | 308);
    }
  }

  // 204/205/304 are "null body status" codes: the Response constructor throws
  // ("Invalid response status code 204") if given any body, even an empty one.
  const NULL_BODY_STATUS = new Set([204, 205, 304]);
  const body = NULL_BODY_STATUS.has(upstream.status) ? null : await upstream.arrayBuffer();
  return new NextResponse(body, {
    status: upstream.status,
    statusText: upstream.statusText,
    headers: responseHeaders,
  });
}

type Params = { params: Promise<{ path: string[] }> };

export async function GET(request: NextRequest, ctx: Params) {
  const { path } = await ctx.params;
  return forward(request, path);
}
export async function POST(request: NextRequest, ctx: Params) {
  const { path } = await ctx.params;
  return forward(request, path);
}
export async function PUT(request: NextRequest, ctx: Params) {
  const { path } = await ctx.params;
  return forward(request, path);
}
export async function PATCH(request: NextRequest, ctx: Params) {
  const { path } = await ctx.params;
  return forward(request, path);
}
export async function DELETE(request: NextRequest, ctx: Params) {
  const { path } = await ctx.params;
  return forward(request, path);
}
export async function HEAD(request: NextRequest, ctx: Params) {
  const { path } = await ctx.params;
  return forward(request, path);
}
export async function OPTIONS(request: NextRequest, ctx: Params) {
  const { path } = await ctx.params;
  return forward(request, path);
}
