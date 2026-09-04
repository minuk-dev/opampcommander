import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { EMPTY_LIST_FILTERS, type ListFilters } from '@shared/lib';
import ListFilterBar from './ListFilterBar';

function setup(value: ListFilters = EMPTY_LIST_FILTERS) {
  const onChange = vi.fn();
  render(<ListFilterBar value={value} onChange={onChange} />);
  return { onChange, user: userEvent.setup() };
}

describe('ListFilterBar', () => {
  // Each applied filter is a new request and a reset to page 0, so applying on
  // every keystroke would fetch once per character.
  it('applies on submit, not on every keystroke', async () => {
    const { onChange, user } = setup();

    await user.type(screen.getByLabelText('Name'), 'otel-');
    expect(onChange).not.toHaveBeenCalled();

    await user.click(screen.getByRole('button', { name: 'Filter' }));
    expect(onChange).toHaveBeenCalledWith({ ...EMPTY_LIST_FILTERS, name: 'otel-' });
  });

  it('passes the label selector through untouched', async () => {
    const { onChange, user } = setup();

    await user.type(screen.getByLabelText('Labels'), 'tier notin (canary,dev)');
    await user.click(screen.getByRole('button', { name: 'Filter' }));

    expect(onChange).toHaveBeenCalledWith({
      ...EMPTY_LIST_FILTERS,
      labelSelector: 'tier notin (canary,dev)',
    });
  });

  it('trims the inputs so a stray space is not sent as a filter', async () => {
    const { onChange, user } = setup();

    await user.type(screen.getByLabelText('Name'), '  otel-  ');
    await user.click(screen.getByRole('button', { name: 'Filter' }));

    expect(onChange).toHaveBeenCalledWith({ ...EMPTY_LIST_FILTERS, name: 'otel-' });
  });

  it('shows a chip per applied filter and clears just that one', async () => {
    const { onChange, user } = setup({
      name: 'otel-',
      labelSelector: 'env=prod',
      fieldSelector: '',
    });

    expect(screen.getByText('Name: otel-')).toBeInTheDocument();
    expect(screen.getByText('Labels: env=prod')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Clear Labels: env=prod' }));
    expect(onChange).toHaveBeenCalledWith({
      name: 'otel-',
      labelSelector: '',
      fieldSelector: '',
    });
  });

  it('clears everything at once', async () => {
    const { onChange, user } = setup({
      name: 'otel-',
      labelSelector: 'env=prod',
      fieldSelector: 'spec.platform=vm',
    });

    await user.click(screen.getByRole('button', { name: 'Clear all' }));
    expect(onChange).toHaveBeenCalledWith(EMPTY_LIST_FILTERS);
  });

  it('renders no chips when nothing is applied', () => {
    setup();
    expect(screen.queryByText('Filters:')).not.toBeInTheDocument();
  });
});
