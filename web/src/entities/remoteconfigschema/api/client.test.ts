import { expect, it, vi } from 'vitest';
import { api } from '@shared/api';
import { listRemoteConfigSchemas } from './client';
vi.mock('@shared/api', () => ({ api: { get: vi.fn() } }));
it('loads every cursor page within the requested namespace', async () => {
  vi.mocked(api.get)
    .mockResolvedValueOnce({ items: [{ metadata: { name: 'a' } }], metadata: { continue: 'next' } })
    .mockResolvedValueOnce({ items: [{ metadata: { name: 'b' } }], metadata: { continue: '' } });
  expect(await listRemoteConfigSchemas('team')).toEqual([
    { metadata: { name: 'a' } },
    { metadata: { name: 'b' } },
  ]);
  expect(api.get).toHaveBeenNthCalledWith(2, '/api/v1/namespaces/team/remoteconfigschemas', {
    query: { limit: 100, continue: 'next' },
  });
});
