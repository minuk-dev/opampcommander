import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { NextRequest } from 'next/server';
import { DELETE, GET } from './route';

function ctx(path: string[]) {
  return { params: Promise.resolve({ path }) };
}

describe('proxy route', () => {
  const fetchSpy = vi.fn();

  beforeEach(() => {
    fetchSpy.mockReset();
    vi.stubGlobal('fetch', fetchSpy);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  // Regression: 204/205/304 are null-body status codes. Forwarding the upstream
  // body (even an empty ArrayBuffer) into the Response constructor throws
  // "Invalid response status code 204". This surfaced when deleting an agent.
  it.each([204, 205, 304])('forwards %i without a body and does not throw', async (status) => {
    fetchSpy.mockResolvedValueOnce(new Response(null, { status }));

    const req = new NextRequest('http://localhost/api/proxy/api/v1/namespaces/default/agents/abc', {
      method: 'DELETE',
    });
    const res = await DELETE(req, ctx(['api', 'v1', 'namespaces', 'default', 'agents', 'abc']));

    expect(res.status).toBe(status);
    expect(res.body).toBeNull();
  });

  it('forwards a normal 200 JSON body through', async () => {
    fetchSpy.mockResolvedValueOnce(
      new Response(JSON.stringify({ ok: true }), {
        status: 200,
        headers: { 'content-type': 'application/json' },
      }),
    );

    const req = new NextRequest('http://localhost/api/proxy/api/v1/agents', { method: 'GET' });
    const res = await GET(req, ctx(['api', 'v1', 'agents']));

    expect(res.status).toBe(200);
    expect(await res.json()).toEqual({ ok: true });
  });
});
