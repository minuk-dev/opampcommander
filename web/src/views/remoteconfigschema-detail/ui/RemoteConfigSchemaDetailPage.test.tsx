import { expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { useApi } from '@shared/api';
import type * as SharedApi from '@shared/api';
import RemoteConfigSchemaDetailPage from './RemoteConfigSchemaDetailPage';
vi.mock('next/navigation', () => ({ useParams: () => ({ namespace: 'team', name: 'contrib' }) }));
vi.mock('@shared/api', async (original) => ({
  ...(await original<typeof SharedApi>()),
  useApi: vi.fn(),
}));
it('groups every component class and reveals full schemas on demand', async () => {
  vi.mocked(useApi).mockReturnValue({
    data: {
      metadata: { name: 'contrib', createdAt: '' },
      spec: {
        binary: 'otelcol',
        version: '1.0',
        components: {
          receivers: {
            otlp: {
              type: 'otlp',
              signals: ['traces'],
              stability: { traces: 'stable' },
              module: 'example/receiver',
              fields: {
                type: 'map',
                children: { endpoint: { type: 'string', doc: 'Listen address' } },
              },
            },
          },
          processors: {},
          exporters: {},
          extensions: {},
          connectors: { forward: { type: 'forward', pairs: [{ from: 'traces', to: 'metrics' }] } },
          custom: {},
        },
      },
    },
    isLoading: false,
  } as ReturnType<typeof useApi>);
  render(<RemoteConfigSchemaDetailPage />);
  for (const name of [
    'receivers (1)',
    'processors (0)',
    'exporters (0)',
    'extensions (0)',
    'connectors (1)',
    'custom (0)',
  ])
    expect(screen.getByRole('heading', { name })).toBeInTheDocument();
  expect(screen.getByText('traces → metrics')).toBeInTheDocument();
  expect(screen.queryByText(/Listen address/)).not.toBeInTheDocument();
  await userEvent.click(screen.getByRole('button', { name: 'Full component: otlp' }));
  expect(screen.getByText(/Listen address/)).toHaveTextContent('example/receiver');
  expect(useApi).toHaveBeenCalledWith('/api/v1/namespaces/team/remoteconfigschemas/contrib');
});

vi.mock('@shared/preferences', () => ({
  TimeDisplay: ({ value }: { value: string }) => <span>{value}</span>,
}));
