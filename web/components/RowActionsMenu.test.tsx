import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import RowActionsMenu from './RowActionsMenu';

// next/navigation isn't wired in the test environment — stub useRouter so the
// component can still render and we can assert against push calls.
const pushSpy = vi.fn();
vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: pushSpy }),
}));

describe('RowActionsMenu', () => {
  it('renders nothing when there are no actions', () => {
    const { container } = render(<RowActionsMenu actions={[]} />);
    expect(container.firstChild).toBeNull();
  });

  it('opens a menu and runs onClick handlers', async () => {
    const user = userEvent.setup();
    const handleClick = vi.fn();
    render(
      <RowActionsMenu actions={[{ label: 'Delete', destructive: true, onClick: handleClick }]} />,
    );

    await user.click(screen.getByRole('button', { name: /actions/i }));
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));

    expect(handleClick).toHaveBeenCalledTimes(1);
  });

  it('navigates via router.push when an action has href', async () => {
    pushSpy.mockClear();
    const user = userEvent.setup();
    render(<RowActionsMenu actions={[{ label: 'View detail', href: '/things/42' }]} />);

    await user.click(screen.getByRole('button', { name: /actions/i }));
    await user.click(await screen.findByRole('menuitem', { name: 'View detail' }));

    expect(pushSpy).toHaveBeenCalledWith('/things/42');
  });
});
