import { beforeEach, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SWRConfig } from 'swr';
import { TooltipProvider } from '@shared/ui';
import { api } from '@shared/api';
import type * as SharedApi from '@shared/api';
import AgentGroupsPage from './AgentGroupsPage';

let agent = 'collector';
vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: vi.fn() }),
  useSearchParams: () => new URLSearchParams({ agent }),
}));
vi.mock('@entities/namespace', () => ({ useNamespace: () => ({ namespace: 'team' }) }));
vi.mock('@shared/preferences', () => ({ TimeDisplay: () => null }));
vi.mock('@shared/api/client', async (original) => ({
  ...(await original<typeof SharedApi>()),
  api: { get: vi.fn(), delete: vi.fn() },
}));
const group = (name: string) => ({
  metadata: { name, namespace: 'team' },
  spec: { priority: 0 },
  status: { numAgents: 1 },
});
beforeEach(() => {
  agent = 'collector';
  vi.clearAllMocks();
  vi.mocked(api.get).mockImplementation((_path, options) => {
    const next = options?.query?.continue;
    return Promise.resolve({
      items: next ? [group('last')] : Array.from({ length: 50 }, (_, i) => group(`group-${i}`)),
      metadata: { continue: next ? 'end' : 'next', remainingItemCount: next ? 0 : 1 },
    });
  });
});
it('navigates membership pages and resets the cursor when the agent changes', async () => {
  const user = userEvent.setup();
  const cache = new Map();
  const view = () => (
    <SWRConfig value={{ provider: () => cache }}>
      <TooltipProvider>
        <AgentGroupsPage />
      </TooltipProvider>
    </SWRConfig>
  );
  const { rerender } = render(view());
  await screen.findByRole('link', { name: 'group-0' });
  expect(screen.getByText('1–50 of 51')).toBeInTheDocument();
  await user.click(screen.getByRole('button', { name: 'Next page' }));
  await screen.findByRole('link', { name: 'last' });
  expect(api.get).toHaveBeenLastCalledWith('/api/v1/namespaces/team/agents/collector/agentgroups', {
    query: { limit: 50, continue: 'next' },
  });
  expect(screen.getByText('51–51 of 51')).toBeInTheDocument();
  expect(screen.queryByRole('link', { name: 'group-0' })).not.toBeInTheDocument();
  expect(screen.getByRole('button', { name: 'Next page' })).toBeDisabled();
  await user.click(screen.getByRole('button', { name: 'Previous page' }));
  await screen.findByRole('link', { name: 'group-0' });
  await user.click(screen.getByRole('button', { name: 'Next page' }));
  await screen.findByRole('link', { name: 'last' });
  agent = 'other';
  rerender(view());
  await screen.findByRole('link', { name: 'group-0' });
  expect(api.get).toHaveBeenLastCalledWith('/api/v1/namespaces/team/agents/other/agentgroups', {
    query: { limit: 50, continue: undefined },
  });
  expect(screen.getByRole('button', { name: 'Previous page' })).toBeDisabled();
});
