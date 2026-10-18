import { beforeEach, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SWRConfig } from 'swr';
import { api } from '@shared/api';
import type * as SharedApi from '@shared/api';
import { toYAML } from '@shared/lib';
import RemoteConfigSchemasPage from './RemoteConfigSchemasPage';

vi.mock('@entities/namespace', () => ({ useNamespace: () => ({ namespace: 'team' }) }));
vi.mock('next/navigation', () => ({ useRouter: () => ({ push: vi.fn() }) }));
vi.mock('@shared/api/client', async (original) => ({
  ...(await original<typeof SharedApi>()),
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
}));
const schema = {
  metadata: { name: 'contrib', namespace: 'team', createdAt: '2026-01-01T00:00:00Z' },
  spec: {
    binary: 'otelcol-contrib',
    version: '0.130.0',
    components: { receivers: { otlp: { type: 'otlp' } }, exporters: { debug: { type: 'debug' } } },
  },
};
beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(api.get).mockResolvedValue({
    items: [schema],
    metadata: { continue: '', remainingItemCount: 0 },
  });
  vi.mocked(api.post).mockResolvedValue(undefined);
  vi.mocked(api.put).mockResolvedValue(undefined);
  vi.mocked(api.delete).mockResolvedValue(undefined);
});
function mount() {
  render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <RemoteConfigSchemasPage />
    </SWRConfig>,
  );
}
it('lists the catalog count and links to a namespace-qualified detail route', async () => {
  mount();
  expect(await screen.findByRole('link', { name: 'contrib' })).toHaveAttribute(
    'href',
    '/remoteconfigschemas/team/contrib',
  );
  expect(screen.getByText('otelcol-contrib')).toBeInTheDocument();
  expect(screen.getByText('0.130.0')).toBeInTheDocument();
  expect(screen.getByRole('cell', { name: '2' })).toBeInTheDocument();
});
it('creates a schema with the shared YAML editor', async () => {
  const user = userEvent.setup();
  mount();
  await user.click(screen.getByRole('button', { name: 'New' }));
  const editor = screen.getByLabelText('editor');
  await user.clear(editor);
  await user.paste(toYAML(schema));
  await user.click(screen.getByRole('button', { name: 'Save' }));
  await waitFor(() =>
    expect(api.post).toHaveBeenCalledWith('/api/v1/namespaces/team/remoteconfigschemas', schema),
  );
});
it('updates a schema without discarding its catalog', async () => {
  const user = userEvent.setup();
  mount();
  await screen.findByText('contrib');
  await user.click(screen.getByRole('button', { name: /actions/i }));
  await user.click(await screen.findByRole('menuitem', { name: 'Edit' }));
  await user.click(screen.getByRole('button', { name: 'Save' }));
  await waitFor(() =>
    expect(api.put).toHaveBeenCalledWith(
      '/api/v1/namespaces/team/remoteconfigschemas/contrib',
      schema,
    ),
  );
});
it('deletes a schema after confirmation', async () => {
  const user = userEvent.setup();
  mount();
  await screen.findByText('contrib');
  await user.click(screen.getByRole('button', { name: /actions/i }));
  await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));
  expect(api.delete).not.toHaveBeenCalled();
  await user.click(screen.getByRole('button', { name: 'Delete' }));
  await waitFor(() =>
    expect(api.delete).toHaveBeenCalledWith(
      '/api/v1/namespaces/team/remoteconfigschemas/contrib',
      undefined,
    ),
  );
});

vi.mock('@shared/preferences', () => ({
  TimeDisplay: ({ value }: { value: string }) => <span>{value}</span>,
}));
