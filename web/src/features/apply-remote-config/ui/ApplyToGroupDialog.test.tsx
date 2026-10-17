import { beforeEach, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { api, useApi } from '@shared/api';
import type { AgentGroup } from '@entities/agent-group';
import type { AgentRemoteConfig } from '@entities/agent-remote-config';
import ApplyToGroupDialog from './ApplyToGroupDialog';

vi.mock('@shared/api', () => ({
  api: { get: vi.fn(), put: vi.fn() },
  useApi: vi.fn(),
}));

const group: AgentGroup = {
  metadata: {
    name: 'collectors',
    namespace: 'original',
    resourceVersion: '3',
    attributes: {},
    createdAt: '2026-01-01T00:00:00Z',
  },
  spec: {
    priority: 1,
    selector: {},
    agentConfig: {
      agentRemoteConfigs: [
        {
          agentRemoteConfigName: 'inline',
          agentRemoteConfigSpec: { value: 'inline', contentType: 'text/plain' },
        },
        { agentRemoteConfigRef: 'existing' },
      ],
    },
  },
  status: {
    numAgents: 1,
    numConnectedAgents: 0,
    numHealthyAgents: 0,
    numUnhealthyAgents: 0,
    numNotConnectedAgents: 1,
  },
};
const config: AgentRemoteConfig = {
  metadata: { name: 'config', namespace: 'original', createdAt: '2026-01-01T00:00:00Z' },
  spec: { value: 'config', contentType: 'text/plain' },
};

function listGroups(items: AgentGroup[]) {
  vi.mocked(useApi).mockReturnValue({ data: { items }, isLoading: false } as ReturnType<
    typeof useApi
  >);
}

beforeEach(() => {
  vi.mocked(api.get).mockReset().mockResolvedValue(group);
  vi.mocked(api.put).mockReset().mockResolvedValue(undefined);
  listGroups([group]);
});

async function selectGroup() {
  const user = userEvent.setup();
  await user.click(screen.getByRole('combobox'));
  await user.click(screen.getByRole('option', { name: group.metadata.name }));
  return user;
}

it('keeps the selected revision, identity and config through refreshes and namespace changes', async () => {
  const props = { open: true, namespace: 'original', config, onClose: vi.fn(), onApplied: vi.fn() };
  const view = render(<ApplyToGroupDialog {...props} />);
  const user = await selectGroup();
  listGroups([
    {
      ...group,
      metadata: { ...group.metadata, resourceVersion: '4' },
      spec: { ...group.spec, priority: 99 },
    },
  ]);
  view.rerender(
    <ApplyToGroupDialog
      {...props}
      namespace="other"
      config={{
        ...config,
        metadata: { ...config.metadata, name: 'other-config', namespace: 'other' },
      }}
    />,
  );
  await user.click(screen.getByRole('button', { name: 'Apply' }));

  expect(api.get).not.toHaveBeenCalled();
  expect(api.put).toHaveBeenCalledExactlyOnceWith(
    '/api/v1/namespaces/original/agentgroups/collectors',
    {
      ...group,
      spec: {
        ...group.spec,
        agentConfig: {
          agentRemoteConfigs: [
            ...group.spec.agentConfig!.agentRemoteConfigs!,
            { agentRemoteConfigRef: 'config' },
          ],
        },
      },
    },
  );
  expect(props.onApplied).toHaveBeenCalledOnce();
});

it('shows conflicts without rereading or retrying against a newer revision', async () => {
  vi.mocked(api.put).mockRejectedValue(
    Object.assign(new Error('The resource was modified concurrently.'), { status: 409 }),
  );
  const onApplied = vi.fn();
  render(
    <ApplyToGroupDialog
      open
      namespace="original"
      config={config}
      onClose={vi.fn()}
      onApplied={onApplied}
    />,
  );
  const user = await selectGroup();
  await user.click(screen.getByRole('button', { name: 'Apply' }));

  expect(
    await screen.findByText(/Close this dialog, refresh the list, and reopen/),
  ).toBeInTheDocument();
  expect(screen.getByText(/The resource was modified concurrently/)).toBeInTheDocument();
  expect(api.put).toHaveBeenCalledOnce();
  expect(api.get).not.toHaveBeenCalled();
  expect(onApplied).not.toHaveBeenCalled();
});

it('clears the old selection and captures the new namespace on reopen', async () => {
  const props = { namespace: 'original', config, onClose: vi.fn(), onApplied: vi.fn() };
  const view = render(<ApplyToGroupDialog {...props} open />);
  const user = await selectGroup();
  view.rerender(<ApplyToGroupDialog {...props} open={false} />);
  const nextGroup = {
    ...group,
    metadata: { ...group.metadata, namespace: 'other', resourceVersion: '8' },
  };
  listGroups([nextGroup]);
  view.rerender(
    <ApplyToGroupDialog
      {...props}
      open
      namespace="other"
      config={{ ...config, metadata: { ...config.metadata, namespace: 'other' } }}
    />,
  );
  expect(screen.getByRole('button', { name: 'Apply' })).toBeDisabled();
  await selectGroup();
  await user.click(screen.getByRole('button', { name: 'Apply' }));
  expect(api.put).toHaveBeenCalledWith(
    '/api/v1/namespaces/other/agentgroups/collectors',
    expect.objectContaining({ metadata: nextGroup.metadata }),
  );
});

it('retries a lost response with the same resource revision and desired state', async () => {
  vi.mocked(api.put).mockRejectedValueOnce(new Error('Network error'));
  const onApplied = vi.fn();
  render(
    <ApplyToGroupDialog
      open
      namespace="original"
      config={config}
      onClose={vi.fn()}
      onApplied={onApplied}
    />,
  );
  const user = await selectGroup();
  await user.click(screen.getByRole('button', { name: 'Apply' }));
  expect(await screen.findByText('Network error')).toBeInTheDocument();
  expect(onApplied).not.toHaveBeenCalled();
  await user.click(screen.getByRole('button', { name: 'Apply' }));
  expect(api.put).toHaveBeenCalledTimes(2);
  expect(vi.mocked(api.put).mock.calls[1]).toEqual(vi.mocked(api.put).mock.calls[0]);
  expect(api.get).not.toHaveBeenCalled();
  expect(onApplied).toHaveBeenCalledOnce();
});
