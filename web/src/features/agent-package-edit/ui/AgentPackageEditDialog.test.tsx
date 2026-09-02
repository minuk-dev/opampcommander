import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import AgentPackageEditDialog from './AgentPackageEditDialog';
import { api } from '@shared/api';
import type * as SharedApi from '@shared/api';
import type { AgentPackage } from '@entities/agent-package';

vi.mock('@shared/api', async (importOriginal) => {
  const actual = await importOriginal<typeof SharedApi>();
  return { ...actual, api: { post: vi.fn(), put: vi.fn() } };
});

const post = vi.mocked(api.post);
const put = vi.mocked(api.put);

beforeEach(() => {
  post.mockReset().mockResolvedValue(undefined as never);
  put.mockReset().mockResolvedValue(undefined as never);
  // Drop any matchMedia stub a previous test installed.
  vi.unstubAllGlobals();
  vi.stubGlobal(
    'fetch',
    vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      statusText: 'OK',
      text: () => Promise.resolve('[]'),
    }),
  );
});

const HASH = 'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=';

const stored: AgentPackage = {
  metadata: {
    name: 'otelcol-linux-amd64',
    namespace: 'default',
    attributes: { team: 'platform' },
    createdAt: '2026-01-01T00:00:00Z',
  },
  spec: {
    packageType: 'TopLevel',
    version: '0.110.0',
    downloadUrl: 'https://example.com/otelcol.tar.gz',
    contentHash: HASH,
  },
};

// MUI decides `fullScreen` through window.matchMedia; jsdom always answers
// "false", so stub a phone-width viewport to exercise the mobile branch.
function stubNarrowViewport() {
  vi.stubGlobal(
    'matchMedia',
    (query: string) =>
      ({
        matches: query.includes('max-width'),
        media: query,
        onchange: null,
        addEventListener: () => {},
        removeEventListener: () => {},
        addListener: () => {},
        removeListener: () => {},
        dispatchEvent: () => false,
      }) as unknown as MediaQueryList,
  );
}

describe('AgentPackageEditDialog', () => {
  it('creates a package from the form fields', async () => {
    const user = userEvent.setup();
    const onSaved = vi.fn();
    render(
      <AgentPackageEditDialog
        open
        mode="create"
        namespace="default"
        onClose={() => {}}
        onSaved={onSaved}
      />,
    );

    await user.type(screen.getByLabelText('Name'), 'otelcol-linux-amd64');
    await user.type(screen.getByLabelText('Version'), '0.110.0');
    await user.type(screen.getByLabelText('Download URL'), 'https://example.com/otelcol.tar.gz');
    await user.type(screen.getByLabelText(/Content hash/), HASH);
    await user.click(screen.getByRole('button', { name: 'Create' }));

    await waitFor(() => expect(post).toHaveBeenCalledTimes(1));
    const [path, body] = post.mock.calls[0];
    expect(path).toBe('/api/v1/namespaces/default/agentpackages');
    expect(body).toMatchObject({
      metadata: { name: 'otelcol-linux-amd64', namespace: 'default' },
      spec: {
        packageType: 'TopLevel',
        version: '0.110.0',
        downloadUrl: 'https://example.com/otelcol.tar.gz',
        contentHash: HASH,
      },
    });
    expect(onSaved).toHaveBeenCalledTimes(1);
  });

  it('leaves optional spec fields out rather than sending empty strings', async () => {
    const user = userEvent.setup();
    render(
      <AgentPackageEditDialog
        open
        mode="create"
        namespace="default"
        onClose={() => {}}
        onSaved={() => {}}
      />,
    );

    await user.type(screen.getByLabelText('Name'), 'minimal');
    await user.type(screen.getByLabelText('Download URL'), 'https://example.com/a.tar.gz');
    await user.click(screen.getByRole('button', { name: 'Create' }));

    await waitFor(() => expect(post).toHaveBeenCalledTimes(1));
    const spec = (post.mock.calls[0][1] as AgentPackage).spec;
    expect(spec).not.toHaveProperty('contentHash');
    expect(spec).not.toHaveProperty('signature');
    expect(spec).not.toHaveProperty('headers');
  });

  it('requires an absolute http(s) download URL', async () => {
    const user = userEvent.setup();
    render(
      <AgentPackageEditDialog
        open
        mode="create"
        namespace="default"
        onClose={() => {}}
        onSaved={() => {}}
      />,
    );

    const create = screen.getByRole('button', { name: 'Create' });
    await user.type(screen.getByLabelText('Name'), 'pkg');
    expect(create).toBeDisabled();

    await user.type(screen.getByLabelText('Download URL'), 'ftp://example.com/a.tgz');
    expect(await screen.findByText('Download URL must use http or https.')).toBeInTheDocument();
    expect(create).toBeDisabled();

    await user.clear(screen.getByLabelText('Download URL'));
    await user.type(screen.getByLabelText('Download URL'), 'https://example.com/a.tgz');
    expect(create).toBeEnabled();
  });

  it('rejects a hex digest pasted into a base64 field', async () => {
    const user = userEvent.setup();
    render(
      <AgentPackageEditDialog
        open
        mode="create"
        namespace="default"
        onClose={() => {}}
        onSaved={() => {}}
      />,
    );

    await user.type(screen.getByLabelText('Name'), 'pkg');
    await user.type(screen.getByLabelText('Download URL'), 'https://example.com/a.tgz');
    await user.type(screen.getByLabelText(/Content hash/), 'deadbeef0');

    expect(
      await screen.findByText(
        /Content hash must be base64 \(a hex digest needs converting first\)/,
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Create' })).toBeDisabled();
  });

  it('loads an existing package and preserves its identity on save', async () => {
    const user = userEvent.setup();
    render(
      <AgentPackageEditDialog
        open
        mode="edit"
        namespace="default"
        initial={stored}
        onClose={() => {}}
        onSaved={() => {}}
      />,
    );

    expect(screen.getByLabelText('Name')).toBeDisabled();
    expect(screen.getByLabelText('Version')).toHaveValue('0.110.0');

    await user.clear(screen.getByLabelText('Version'));
    await user.type(screen.getByLabelText('Version'), '0.111.0');
    await user.click(screen.getByRole('button', { name: 'Save changes' }));

    await waitFor(() => expect(put).toHaveBeenCalledTimes(1));
    const [path, body] = put.mock.calls[0];
    expect(path).toBe('/api/v1/namespaces/default/agentpackages/otelcol-linux-amd64');
    expect(body).toMatchObject({
      metadata: {
        name: 'otelcol-linux-amd64',
        namespace: 'default',
        attributes: { team: 'platform' },
        createdAt: stored.metadata.createdAt,
      },
      spec: { version: '0.111.0', contentHash: HASH },
    });
  });

  it('goes full screen on a phone-width viewport', () => {
    stubNarrowViewport();
    render(
      <AgentPackageEditDialog
        open
        mode="create"
        namespace="default"
        onClose={() => {}}
        onSaved={() => {}}
      />,
    );

    expect(document.querySelector('.MuiDialog-paperFullScreen')).not.toBeNull();
    expect(screen.getByLabelText('close')).toBeInTheDocument();
  });

  it('surfaces the server problem detail when the save is rejected', async () => {
    const user = userEvent.setup();
    post.mockRejectedValue(
      Object.assign(new Error('package already exists'), {
        status: 409,
        body: { title: 'Conflict', detail: 'package already exists' },
      }),
    );

    render(
      <AgentPackageEditDialog
        open
        mode="create"
        namespace="default"
        onClose={() => {}}
        onSaved={() => {}}
      />,
    );

    await user.type(screen.getByLabelText('Name'), 'dup');
    await user.type(screen.getByLabelText('Download URL'), 'https://example.com/a.tgz');
    await user.click(screen.getByRole('button', { name: 'Create' }));

    expect(await screen.findByText('package already exists')).toBeInTheDocument();
    expect(screen.getByText(/Conflict · HTTP 409/)).toBeInTheDocument();
  });
});
