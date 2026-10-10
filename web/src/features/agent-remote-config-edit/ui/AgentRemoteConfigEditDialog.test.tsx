import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import AgentRemoteConfigEditDialog from './AgentRemoteConfigEditDialog';
import { api } from '@shared/api';
import type * as SharedApi from '@shared/api';
import { SWRConfig } from 'swr';
import type { AgentRemoteConfig } from '@entities/agent-remote-config';

vi.mock('@shared/api', async (importOriginal) => {
  const actual = await importOriginal<typeof SharedApi>();
  return { ...actual, api: { get: vi.fn(), post: vi.fn(), put: vi.fn() } };
});

const post = vi.mocked(api.post);
const put = vi.mocked(api.put);

// The dialog loads its sample menu over fetch; an empty list keeps it quiet.
beforeEach(() => {
  vi.mocked(api.get)
    .mockReset()
    .mockResolvedValue({ items: [], metadata: { continue: '' } });
  post.mockReset().mockResolvedValue(undefined);
  put.mockReset().mockResolvedValue(undefined);
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

const stored: AgentRemoteConfig = {
  metadata: {
    resourceVersion: '3',
    name: 'otlp-debug',
    namespace: 'default',
    attributes: { team: 'platform' },
    createdAt: '2026-01-01T00:00:00Z',
  },
  spec: {
    value: 'exporters:\n  debug:\n    verbosity: basic\n',
    contentType: 'text/yaml',
    schemaRefs: ['otelcol-0.110'],
  },
  status: { schemaRefsSource: 'auto' },
};

// The body editor is lazily imported, so wait for the real textarea to replace
// the loading fallback.
async function bodyEditor() {
  return await screen.findByLabelText('config body');
}

describe('AgentRemoteConfigEditDialog', () => {
  it('creates a config from the name, content type and body', async () => {
    const user = userEvent.setup();
    const onSaved = vi.fn();
    render(
      <AgentRemoteConfigEditDialog
        open
        mode="create"
        namespace="default"
        onClose={() => {}}
        onSaved={onSaved}
      />,
    );

    await user.type(screen.getByLabelText('Name'), 'otlp-debug');
    await user.type(await bodyEditor(), 'receivers:{enter}  otlp: null');
    await user.click(screen.getByRole('button', { name: 'Create' }));

    await waitFor(() => expect(post).toHaveBeenCalledTimes(1));
    const [path, body] = post.mock.calls[0];
    expect(path).toBe('/api/v1/namespaces/default/agentremoteconfigs');
    expect(body).toMatchObject({
      metadata: { name: 'otlp-debug', namespace: 'default' },
      spec: { value: 'receivers:\n  otlp: null', contentType: 'text/yaml' },
    });
    expect(onSaved).toHaveBeenCalledTimes(1);
  });

  it('blocks saving while the body is not valid YAML and points at the line', async () => {
    const user = userEvent.setup();
    render(
      <AgentRemoteConfigEditDialog
        open
        mode="create"
        namespace="default"
        onClose={() => {}}
        onSaved={() => {}}
      />,
    );

    await user.type(screen.getByLabelText('Name'), 'broken');
    await user.type(await bodyEditor(), 'a: 1{enter}a: 2');

    expect(await screen.findByText(/Invalid YAML \(line 2\)/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Create' })).toBeDisabled();
    expect(post).not.toHaveBeenCalled();
  });

  it('does not YAML-validate a body whose content type is not YAML', async () => {
    const user = userEvent.setup();
    render(
      <AgentRemoteConfigEditDialog
        open
        mode="create"
        namespace="default"
        onClose={() => {}}
        onSaved={() => {}}
      />,
    );

    await user.type(screen.getByLabelText('Name'), 'plain');
    await user.click(screen.getByLabelText('Content type'));
    await user.click(screen.getByRole('option', { name: 'text/plain' }));
    await user.type(await bodyEditor(), 'a: 1{enter}a: 2');

    expect(screen.queryByText(/Invalid YAML/)).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Create' })).toBeEnabled();
  });

  it('preserves stored schema references when saving an existing config', async () => {
    const user = userEvent.setup();
    render(
      <AgentRemoteConfigEditDialog
        open
        mode="edit"
        namespace="default"
        initial={stored}
        onClose={() => {}}
        onSaved={() => {}}
      />,
    );

    const editor = await bodyEditor();
    expect(screen.getByText('Auto-resolved')).toBeInTheDocument();
    expect(editor).toHaveValue(stored.spec.value);
    await user.clear(editor);
    await user.type(editor, 'exporters:{enter}  debug:{enter}  verbosity: detailed');
    await user.click(screen.getByRole('button', { name: /Save/ }));

    await waitFor(() => expect(put).toHaveBeenCalledTimes(1));
    const [path, body] = put.mock.calls[0];
    expect(path).toBe('/api/v1/namespaces/default/agentremoteconfigs/otlp-debug');
    expect(body).toMatchObject({
      metadata: {
        name: 'otlp-debug',
        attributes: { team: 'platform' },
        createdAt: stored.metadata.createdAt,
      },
      spec: { schemaRefs: ['otelcol-0.110'], contentType: 'text/yaml' },
    });
  });

  it('keeps Save disabled until something actually changes', async () => {
    const user = userEvent.setup();
    render(
      <AgentRemoteConfigEditDialog
        open
        mode="edit"
        namespace="default"
        initial={stored}
        onClose={() => {}}
        onSaved={() => {}}
      />,
    );

    const editor = await bodyEditor();
    const save = screen.getByRole('button', { name: 'Save changes' });
    expect(save).toBeDisabled();

    await user.type(editor, '# touched{enter}');
    expect(save).toBeEnabled();
  });

  it('shows what the save will change', async () => {
    const user = userEvent.setup();
    render(
      <AgentRemoteConfigEditDialog
        open
        mode="edit"
        namespace="default"
        initial={stored}
        onClose={() => {}}
        onSaved={() => {}}
      />,
    );

    const editor = await bodyEditor();
    await user.clear(editor);
    await user.type(editor, 'exporters:{enter}  debug:{enter}    verbosity: detailed');
    await user.click(screen.getByRole('tab', { name: 'Diff' }));

    expect(await screen.findByText('+1')).toBeInTheDocument();
    expect(screen.getByText('−1')).toBeInTheDocument();
  });

  it('renders a full-bleed panel on phones and a close control', () => {
    render(
      <AgentRemoteConfigEditDialog
        open
        mode="create"
        namespace="default"
        onClose={() => {}}
        onSaved={() => {}}
      />,
    );

    // Below `sm` the dialog covers the viewport; from `sm` up it becomes a
    // centred card. That switch is pure CSS now, so assert the contract on the
    // panel rather than stubbing a viewport jsdom cannot lay out anyway.
    const panel = screen.getByRole('dialog');
    expect(panel.className).toContain('inset-0');
    expect(panel.className).toContain('sm:inset-auto');
    expect(screen.getByLabelText('close')).toBeInTheDocument();
  });

  it('surfaces the server problem detail when the save is rejected', async () => {
    const user = userEvent.setup();
    const err = Object.assign(new Error('config body is empty'), {
      status: 400,
      body: { title: 'Bad Request', detail: 'config body is empty' },
    });
    put.mockRejectedValue(err);

    render(
      <AgentRemoteConfigEditDialog
        open
        mode="edit"
        namespace="default"
        initial={stored}
        onClose={() => {}}
        onSaved={() => {}}
      />,
    );

    const editor = await bodyEditor();
    await user.clear(editor);
    await user.click(screen.getByRole('button', { name: /Save/ }));

    expect(await screen.findByText('config body is empty')).toBeInTheDocument();
    expect(screen.getByText(/Bad Request · HTTP 400/)).toBeInTheDocument();
  });
  it('pins identity and revision while the editor is open', async () => {
    const user = userEvent.setup();
    const props = {
      open: true,
      mode: 'edit' as const,
      namespace: 'default',
      initial: stored,
      onClose: vi.fn(),
      onSaved: vi.fn(),
    };
    const { rerender } = render(<AgentRemoteConfigEditDialog {...props} />);
    await user.type(await bodyEditor(), '# changed{enter}');
    rerender(
      <AgentRemoteConfigEditDialog
        {...props}
        namespace="other"
        initial={{
          ...stored,
          metadata: {
            ...stored.metadata,
            namespace: 'other',
            name: 'replacement',
            resourceVersion: '4',
          },
        }}
      />,
    );
    await user.click(screen.getByRole('button', { name: 'Save changes' }));
    await waitFor(() => expect(put).toHaveBeenCalledTimes(1));
    expect(put.mock.calls[0][0]).toBe(
      `/api/v1/namespaces/default/agentremoteconfigs/${stored.metadata.name}`,
    );
    expect(put.mock.calls[0][1]).toMatchObject({
      metadata: { name: stored.metadata.name, namespace: 'default', resourceVersion: '3' },
    });
  });
});

it('selects multiple schemas, removes an unavailable ref, and changes the skip annotation without losing attributes', async () => {
  const user = userEvent.setup();
  vi.mocked(api.get).mockResolvedValue({
    items: [
      { metadata: { name: 'schema-a' }, spec: { binary: 'otelcol', version: '1.0' } },
      { metadata: { name: 'schema-b' }, spec: { binary: 'otelcol', version: '2.0' } },
    ],
    metadata: { continue: '' },
  });
  render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <AgentRemoteConfigEditDialog
        open
        mode="edit"
        namespace="team"
        initial={{
          ...stored,
          metadata: {
            ...stored.metadata,
            namespace: 'team',
            attributes: { team: 'platform', 'opampcommander.io/skip-schema-validation': 'TRUE' },
          },
        }}
        onClose={() => {}}
        onSaved={() => {}}
      />
    </SWRConfig>,
  );
  expect(screen.getByRole('switch', { name: 'Skip schema validation' })).toBeChecked();
  await user.click(await screen.findByRole('checkbox', { name: /schema-a/ }));
  await user.click(screen.getByRole('checkbox', { name: /schema-b/ }));
  await user.click(screen.getByRole('checkbox', { name: /otelcol-0.110/ }));
  await user.click(screen.getByRole('switch', { name: 'Skip schema validation' }));
  expect(screen.getByText('Explicitly set')).toBeInTheDocument();
  await user.click(screen.getByRole('button', { name: 'Save changes' }));
  await waitFor(() => expect(put).toHaveBeenCalledTimes(1));
  expect(put.mock.calls[0]).toEqual([
    '/api/v1/namespaces/team/agentremoteconfigs/otlp-debug',
    expect.objectContaining({
      metadata: expect.objectContaining({ attributes: { team: 'platform' } }),
      spec: expect.objectContaining({ schemaRefs: ['schema-a', 'schema-b'] }),
    }),
  ]);
  expect(api.get).toHaveBeenCalledWith(
    '/api/v1/namespaces/team/remoteconfigschemas',
    expect.anything(),
  );
});

it('saves the skip annotation when it is the only change', async () => {
  const user = userEvent.setup();
  render(
    <AgentRemoteConfigEditDialog
      open
      mode="edit"
      namespace="default"
      initial={stored}
      onClose={() => {}}
      onSaved={() => {}}
    />,
  );
  await user.click(screen.getByRole('switch', { name: 'Skip schema validation' }));
  await user.click(screen.getByRole('button', { name: 'Save changes' }));
  await waitFor(() => expect(put).toHaveBeenCalledTimes(1));
  expect(put.mock.calls[0][1]).toMatchObject({
    metadata: {
      attributes: { team: 'platform', 'opampcommander.io/skip-schema-validation': 'true' },
    },
  });
});

it('shows a schema loading failure while allowing existing refs to be removed', async () => {
  vi.mocked(api.get).mockRejectedValue(new Error('Forbidden'));
  const user = userEvent.setup();
  render(
    <SWRConfig value={{ provider: () => new Map(), shouldRetryOnError: false }}>
      <AgentRemoteConfigEditDialog
        open
        mode="edit"
        namespace="default"
        initial={stored}
        onClose={() => {}}
        onSaved={() => {}}
      />
    </SWRConfig>,
  );
  expect(await screen.findByText('Failed to load schemas: Forbidden')).toBeInTheDocument();
  await user.click(screen.getByRole('checkbox', { name: /otelcol-0.110/ }));
  await user.click(screen.getByRole('button', { name: 'Save changes' }));
  await waitFor(() => expect(put).toHaveBeenCalledTimes(1));
  expect(put.mock.calls[0][1]).toMatchObject({ spec: { schemaRefs: [] } });
});
