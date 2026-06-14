import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { api, type ApiError } from './client';

const TOKEN_KEY = 'opamp.token';
const REFRESH_KEY = 'opamp.refreshToken';

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'content-type': 'application/json' },
  });
}

describe('api-client', () => {
  const fetchSpy = vi.fn();

  beforeEach(() => {
    fetchSpy.mockReset();
    vi.stubGlobal('fetch', fetchSpy);
    // jsdom gives us window/localStorage; reset between tests
    window.localStorage.clear();
    // Default origin is http://localhost — keep stable across tests
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('GET attaches Bearer token from localStorage and parses JSON', async () => {
    window.localStorage.setItem(TOKEN_KEY, 'abc');
    fetchSpy.mockResolvedValueOnce(jsonResponse({ items: [1, 2] }));

    const result = await api.get<{ items: number[] }>('/api/v1/things');

    expect(result).toEqual({ items: [1, 2] });
    expect(fetchSpy).toHaveBeenCalledTimes(1);
    const [url, init] = fetchSpy.mock.calls[0];
    expect(String(url)).toContain('/api/proxy/api/v1/things');
    expect((init as RequestInit).headers).toMatchObject({
      Authorization: 'Bearer abc',
    });
  });

  it('attaches Basic auth when basicAuth option is set', async () => {
    fetchSpy.mockResolvedValueOnce(jsonResponse({ token: 't' }));

    await api.get('/api/v1/auth/basic', {
      basicAuth: { username: 'alice', password: 's3cret' },
      noAuthRedirect: true,
    });

    const init = fetchSpy.mock.calls[0][1] as RequestInit;
    expect((init.headers as Record<string, string>).Authorization).toBe(
      `Basic ${btoa('alice:s3cret')}`,
    );
  });

  it('appends query params and skips empty values', async () => {
    fetchSpy.mockResolvedValueOnce(jsonResponse({}));

    await api.get('/api/v1/things', {
      query: { limit: 50, q: 'foo', continue: '', skip: null, none: undefined },
    });

    const url = new URL(String(fetchSpy.mock.calls[0][0]));
    expect(url.searchParams.get('limit')).toBe('50');
    expect(url.searchParams.get('q')).toBe('foo');
    expect(url.searchParams.has('continue')).toBe(false);
    expect(url.searchParams.has('skip')).toBe(false);
    expect(url.searchParams.has('none')).toBe(false);
  });

  it('throws an ApiError with status + detail for non-2xx JSON responses', async () => {
    window.localStorage.setItem(TOKEN_KEY, 't');
    fetchSpy.mockResolvedValueOnce(jsonResponse({ detail: 'not found' }, 404));

    const err = await api.get('/api/v1/missing').catch((e: ApiError) => e);
    expect(err).toBeInstanceOf(Error);
    expect((err as ApiError).status).toBe(404);
    expect((err as ApiError).message).toBe('not found');
  });

  it('attempts refresh on 401 and retries with the new token', async () => {
    window.localStorage.setItem(TOKEN_KEY, 'old');
    window.localStorage.setItem(REFRESH_KEY, 'rrr');

    // The backend-path calls happen in order: 401 → refresh(200) → retry(200).
    // A token rotation also fires a best-effort POST /api/session to sync the
    // httpOnly cookie; route that separately so it doesn't consume the
    // backend-path sequence.
    const backendResponses = [
      new Response('{}', { status: 401 }),
      jsonResponse({ token: 'new', refreshToken: 'rrr2' }),
      jsonResponse({ ok: true }),
    ];
    let i = 0;
    fetchSpy.mockImplementation((url: string | URL) => {
      if (String(url).includes('/api/session')) {
        return Promise.resolve(jsonResponse({ ok: true }));
      }
      return Promise.resolve(backendResponses[i++]);
    });

    const result = await api.get<{ ok: boolean }>('/api/v1/secure');
    expect(result).toEqual({ ok: true });

    // localStorage updated to rotated tokens
    expect(window.localStorage.getItem(TOKEN_KEY)).toBe('new');
    expect(window.localStorage.getItem(REFRESH_KEY)).toBe('rrr2');

    // the retry to the secure endpoint used the new bearer
    const secureCalls = fetchSpy.mock.calls.filter((c) => String(c[0]).includes('/api/v1/secure'));
    const retryInit = secureCalls[1][1] as RequestInit;
    expect((retryInit.headers as Record<string, string>).Authorization).toBe('Bearer new');

    // and the session cookie was synced with the rotated token
    expect(fetchSpy.mock.calls.some((c) => String(c[0]).includes('/api/session'))).toBe(true);
  });

  it('does NOT redirect on 401 when noAuthRedirect is set', async () => {
    window.localStorage.setItem(TOKEN_KEY, 'old');
    // no refresh token so refresh path bails out
    fetchSpy.mockResolvedValueOnce(new Response('{}', { status: 401 }));

    const replaceSpy = vi.fn();
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { ...window.location, replace: replaceSpy, pathname: '/agents' },
    });

    const err = await api
      .get('/api/v1/auth/info', { noAuthRedirect: true })
      .catch((e: ApiError) => e);
    expect((err as ApiError).status).toBe(401);
    expect(replaceSpy).not.toHaveBeenCalled();
  });
});
