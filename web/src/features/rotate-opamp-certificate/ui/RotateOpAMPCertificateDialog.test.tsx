import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { api, useApi } from '@shared/api';
import type { AgentGroup } from '@entities/agent-group';
import RotateOpAMPCertificateDialog from './RotateOpAMPCertificateDialog';

vi.mock('@shared/api', () => ({
  api: { get: vi.fn(), put: vi.fn() },
  useApi: vi.fn(),
}));

const group: AgentGroup = {
  metadata: { namespace: 'default', name: 'singleton', attributes: {}, createdAt: '' },
  spec: {
    priority: 0,
    selector: {},
    agentConfig: {
      agentRemoteConfigs: [{ agentRemoteConfigRef: 'existing-config' }],
      connectionSettings: {
        opamp: {
          destinationEndpoint: 'wss://example.test/api/v1/opamp',
          headers: { Authorization: ['old-token'] },
          certificateName: 'old-cert',
        },
        ownMetrics: { destinationEndpoint: 'https://metrics.example.test' },
      },
    },
  },
  status: {
    numAgents: 1,
    numConnectedAgents: 1,
    numHealthyAgents: 1,
    numUnhealthyAgents: 0,
    numNotConnectedAgents: 0,
  },
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.put).mockReset().mockResolvedValue(undefined);
  vi.mocked(useApi)
    .mockReset()
    .mockReturnValue({
      data: {
        items: [{ metadata: { instanceUid: 'agent-uid' } }],
        metadata: { continue: 'cursor', remainingItemCount: 0 },
      },
      error: undefined,
      isLoading: false,
    } as ReturnType<typeof useApi>);
});

describe('RotateOpAMPCertificateDialog', () => {
  it('updates only the OpAMP certificate reference', async () => {
    const user = userEvent.setup();
    const onApplied = vi.fn();
    vi.mocked(api.get)
      .mockResolvedValueOnce(group)
      .mockResolvedValueOnce({
        items: [{ metadata: { instanceUid: 'agent-uid' } }],
        metadata: { remainingItemCount: 0 },
      });

    render(
      <RotateOpAMPCertificateDialog
        namespace="default"
        group={group}
        onClose={() => {}}
        onApplied={onApplied}
      />,
    );

    const input = screen.getByLabelText('Certificate resource name');
    await user.clear(input);
    await user.type(input, 'new-cert');
    await user.click(screen.getByRole('button', { name: 'Offer certificate' }));

    await waitFor(() => expect(api.put).toHaveBeenCalledTimes(1));
    const [path, body] = vi.mocked(api.put).mock.calls[0];
    expect(path).toBe('/api/v1/namespaces/default/agentgroups/singleton');
    expect(body).toMatchObject({
      spec: {
        agentConfig: {
          agentRemoteConfigs: [{ agentRemoteConfigRef: 'existing-config' }],
          connectionSettings: {
            opamp: {
              destinationEndpoint: 'wss://example.test/api/v1/opamp',
              headers: { Authorization: ['old-token'] },
              certificateName: 'new-cert',
            },
            ownMetrics: { destinationEndpoint: 'https://metrics.example.test' },
          },
        },
      },
    });
    expect(onApplied).toHaveBeenCalledOnce();
  });

  it('blocks rotation when the group has multiple agents', () => {
    vi.mocked(useApi).mockReturnValue({
      data: { items: [{}, {}], metadata: { continue: 'cursor', remainingItemCount: 0 } },
      error: undefined,
      isLoading: false,
    } as ReturnType<typeof useApi>);

    render(
      <RotateOpAMPCertificateDialog
        namespace="default"
        group={group}
        onClose={() => {}}
        onApplied={() => {}}
      />,
    );

    expect(screen.getByRole('button', { name: 'Offer certificate' })).toBeDisabled();
    expect(screen.getByText(/requires exactly one agent/)).toBeInTheDocument();
    const panel = screen.getByRole('dialog');
    expect(panel.className).toContain('inset-0');
    expect(panel.className).toContain('sm:inset-auto');
  });

  it('blocks a save when group membership changed after opening the dialog', async () => {
    const user = userEvent.setup();
    vi.mocked(api.get)
      .mockResolvedValueOnce(group)
      .mockResolvedValueOnce({
        items: [{ metadata: { instanceUid: 'another-agent' } }],
        metadata: { remainingItemCount: 0 },
      });

    render(
      <RotateOpAMPCertificateDialog
        namespace="default"
        group={group}
        onClose={() => {}}
        onApplied={() => {}}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Offer certificate' }));

    expect(await screen.findByText(/Group membership changed/)).toBeInTheDocument();
    expect(api.put).not.toHaveBeenCalled();
  });
});
