import { act, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { api } from '@shared/api';
import AgentGroupPicker from './AgentGroupPicker';

vi.mock('@shared/api', () => ({ api: { get: vi.fn() } }));

function page(names: string[], next = '', remaining = 0) {
  return {
    items: names.map((name) => ({ metadata: { name } })),
    metadata: { continue: next, remainingItemCount: remaining },
  };
}

async function debounce() {
  await act(() => vi.advanceTimersByTimeAsync(250));
}

function setup() {
  const onChange = vi.fn();
  const rendered = render(
    <AgentGroupPicker namespace="team" value="" onChange={onChange} id="group" />,
  );
  fireEvent.click(screen.getByRole('combobox'));
  return { ...rendered, onChange };
}

describe('AgentGroupPicker', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.mocked(api.get).mockReset();
  });
  afterEach(() => vi.useRealTimers());

  it('debounces indexed prefix searches and selects a group beyond the first 500 by keyboard', async () => {
    vi.mocked(api.get).mockResolvedValue(page(['group-99999']));
    const { onChange } = setup();
    const input = screen.getByPlaceholderText('Agent group name starts with…');
    fireEvent.change(input, { target: { value: 'group-9' } });
    await act(() => vi.advanceTimersByTimeAsync(150));
    fireEvent.change(input, { target: { value: 'group-99999' } });
    expect(api.get).not.toHaveBeenCalled();
    expect(screen.getByRole('status')).toHaveTextContent('Loading');
    await debounce();
    expect(api.get).toHaveBeenCalledExactlyOnceWith(
      '/api/v1/namespaces/team/agentgroups',
      expect.objectContaining({ query: { name: 'group-99999', limit: 50, continue: '' } }),
    );
    expect(screen.getByRole('option', { name: 'group-99999' })).toBeInTheDocument();
    fireEvent.keyDown(input, { key: 'ArrowDown' });
    fireEvent.keyDown(input, { key: 'Enter' });
    expect(onChange).toHaveBeenCalledWith('group-99999');
  });

  it('uses the resume cursor and replaces the previous page instead of accumulating results', async () => {
    vi.mocked(api.get).mockResolvedValueOnce(page(['first'], 'next', 100000));
    setup();
    await debounce();
    vi.mocked(api.get).mockResolvedValueOnce(page(['last'], 'terminal', 0));
    fireEvent.click(screen.getByRole('button', { name: 'Next results' }));
    await debounce();
    expect(api.get).toHaveBeenLastCalledWith(
      expect.any(String),
      expect.objectContaining({ query: { name: '', limit: 50, continue: 'next' } }),
    );
    expect(screen.queryByRole('option', { name: 'first' })).not.toBeInTheDocument();
    expect(screen.getByRole('option', { name: 'last' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Next results' })).not.toBeInTheDocument();
  });

  it('aborts stale searches and unmounted requests, ignoring late responses', async () => {
    let resolveOld!: (data: ReturnType<typeof page>) => void;
    vi.mocked(api.get).mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          resolveOld = resolve;
        }),
    );
    const { unmount } = setup();
    await debounce();
    const oldSignal = vi.mocked(api.get).mock.calls[0][1]?.signal;
    fireEvent.change(screen.getByPlaceholderText('Agent group name starts with…'), {
      target: { value: 'new' },
    });
    expect(oldSignal?.aborted).toBe(true);
    vi.mocked(api.get).mockResolvedValueOnce(page(['new-group']));
    await debounce();
    await act(() => Promise.resolve(resolveOld(page(['old-group']))));
    expect(screen.queryByRole('option', { name: 'old-group' })).not.toBeInTheDocument();
    expect(screen.getByRole('option', { name: 'new-group' })).toBeInTheDocument();
    const newSignal = vi.mocked(api.get).mock.calls[1][1]?.signal;
    unmount();
    expect(newSignal?.aborted).toBe(true);
  });

  it('shows errors with retry and distinguishes an empty result', async () => {
    vi.mocked(api.get).mockRejectedValueOnce(new Error('Permission denied'));
    setup();
    await debounce();
    expect(screen.getByRole('alert')).toHaveTextContent('Permission denied');
    vi.mocked(api.get).mockResolvedValueOnce(page([]));
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
    await debounce();
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    expect(screen.getByText('No agent groups found')).toBeInTheDocument();
  });

  it('reports a broken continuation instead of looping through the same page', async () => {
    vi.mocked(api.get).mockResolvedValueOnce(page(['first'], '', 10));
    setup();
    await debounce();
    expect(screen.getByRole('alert')).toHaveTextContent('cursor did not advance');
    expect(screen.queryByRole('button', { name: 'Next results' })).not.toBeInTheDocument();
  });
});
