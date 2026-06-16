import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import UserCreateDialog from './UserCreateDialog';
import { api } from '@shared/api';

vi.mock('@shared/api', () => ({
  api: { post: vi.fn() },
}));

const post = vi.mocked(api.post);

describe('UserCreateDialog', () => {
  beforeEach(() => {
    post.mockReset();
    post.mockResolvedValue(undefined as never);
  });

  it('posts the entered password in spec.password when creating a basic-auth user', async () => {
    const user = userEvent.setup();
    const onSaved = vi.fn();
    render(<UserCreateDialog open onClose={() => {}} onSaved={onSaved} />);

    await user.type(screen.getByLabelText(/Email/), 'bob@example.com');
    await user.type(screen.getByLabelText(/Username/), 'bob');
    await user.type(screen.getByLabelText(/^Password/), 'change-me');
    await user.click(screen.getByRole('button', { name: 'Create' }));

    expect(post).toHaveBeenCalledTimes(1);
    const [path, body] = post.mock.calls[0];
    expect(path).toBe('/api/v1/users');
    expect(body).toMatchObject({
      kind: 'User',
      spec: { email: 'bob@example.com', username: 'bob', isActive: true, password: 'change-me' },
    });
    expect(onSaved).toHaveBeenCalledTimes(1);
  });

  it('omits spec.password when no password is entered', async () => {
    const user = userEvent.setup();
    render(<UserCreateDialog open onClose={() => {}} onSaved={() => {}} />);

    await user.type(screen.getByLabelText(/Email/), 'alice@example.com');
    await user.type(screen.getByLabelText(/Username/), 'alice');
    await user.click(screen.getByRole('button', { name: 'Create' }));

    const [, body] = post.mock.calls[0];
    expect(body).toMatchObject({ spec: { email: 'alice@example.com', username: 'alice' } });
    expect((body as { spec: Record<string, unknown> }).spec).not.toHaveProperty('password');
  });

  it('disables Create until both email and username are provided', async () => {
    const user = userEvent.setup();
    render(<UserCreateDialog open onClose={() => {}} onSaved={() => {}} />);

    const create = screen.getByRole('button', { name: 'Create' });
    expect(create).toBeDisabled();

    await user.type(screen.getByLabelText(/Email/), 'alice@example.com');
    expect(create).toBeDisabled();

    await user.type(screen.getByLabelText(/Username/), 'alice');
    expect(create).toBeEnabled();
  });

  it('toggles password visibility', async () => {
    const user = userEvent.setup();
    render(<UserCreateDialog open onClose={() => {}} onSaved={() => {}} />);

    const password = screen.getByLabelText(/^Password/);
    expect(password).toHaveAttribute('type', 'password');

    await user.click(screen.getByRole('button', { name: 'Show password' }));
    expect(password).toHaveAttribute('type', 'text');
  });
});
