import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ConfirmDialog from './ConfirmDialog';

describe('ConfirmDialog', () => {
  it('runs onConfirm when the user clicks the confirm button', async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();
    render(
      <ConfirmDialog
        open
        title="Delete?"
        message="Are you sure?"
        confirmLabel="Delete"
        onClose={() => {}}
        onConfirm={onConfirm}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Delete' }));
    expect(onConfirm).toHaveBeenCalledTimes(1);
  });

  it('disables both buttons while the confirm callback is in flight', async () => {
    const user = userEvent.setup();
    let resolve!: () => void;
    const inFlight = new Promise<void>((r) => {
      resolve = r;
    });

    render(
      <ConfirmDialog
        open
        title="Delete?"
        message="Are you sure?"
        confirmLabel="Delete"
        onClose={() => {}}
        onConfirm={() => inFlight}
      />,
    );

    const confirm = screen.getByRole('button', { name: 'Delete' });
    const cancel = screen.getByRole('button', { name: 'Cancel' });
    await user.click(confirm);

    // While the promise is still pending, both buttons should be disabled so
    // a double-click can't fire a second delete.
    expect(confirm).toBeDisabled();
    expect(cancel).toBeDisabled();

    resolve();
    await inFlight;
  });

  it('shows an inline error and leaves the dialog open when onConfirm throws', async () => {
    const user = userEvent.setup();
    render(
      <ConfirmDialog
        open
        title="Delete?"
        message="Are you sure?"
        confirmLabel="Delete"
        onClose={() => {}}
        onConfirm={() => {
          throw new Error('server exploded');
        }}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Delete' }));
    expect(await screen.findByText('server exploded')).toBeInTheDocument();
    // Dialog still rendered, user can retry or cancel
    expect(screen.getByRole('button', { name: 'Delete' })).not.toBeDisabled();
  });
});
