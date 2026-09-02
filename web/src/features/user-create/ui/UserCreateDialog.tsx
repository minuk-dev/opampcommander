'use client';

import { Eye, EyeOff } from 'lucide-react';
import { useState } from 'react';
import { api } from '@shared/api';
import {
  Alert,
  Button,
  Dialog,
  DialogBody,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Field,
  Input,
} from '@shared/ui';
import type { User } from '@entities/user';

interface Props {
  open: boolean;
  onClose: () => void;
  onSaved: () => void;
}

export default function UserCreateDialog({ open, onClose, onSaved }: Props) {
  const [email, setEmail] = useState('');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const save = async () => {
    setBusy(true);
    setError(null);
    try {
      const now = new Date().toISOString();
      const body: User = {
        kind: 'User',
        apiVersion: 'v1',
        metadata: { uid: '', createdAt: now, updatedAt: now },
        // CreateUser always provisions an active user (spec.isActive is ignored
        // on POST). Password is optional: set it to enable basic
        // (username/password) login. It is write-only — the server stores only a
        // one-way hash and never returns it — so we send it only when non-empty.
        spec: {
          email,
          username,
          isActive: true,
          ...(password ? { password } : {}),
        },
      };
      await api.post('/api/v1/users', body);
      onSaved();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create user');
    } finally {
      setBusy(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent size="sm">
        <DialogHeader>
          <DialogTitle>Create user</DialogTitle>
        </DialogHeader>
        <DialogBody className="space-y-3">
          {error && <Alert severity="error">{error}</Alert>}
          <Field label="Email" required>
            {(field) => (
              <Input
                {...field}
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            )}
          </Field>
          <Field label="Username" required>
            {(field) => (
              <Input {...field} value={username} onChange={(e) => setUsername(e.target.value)} />
            )}
          </Field>
          <Field
            label="Password"
            hint="Optional. Set it to enable basic (username/password) login. Stored only as a one-way hash and never returned."
          >
            {(field) => (
              <Input
                {...field}
                type={showPassword ? 'text' : 'password'}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                autoComplete="new-password"
                endSlot={
                  <button
                    type="button"
                    aria-label={showPassword ? 'Hide password' : 'Show password'}
                    onClick={() => setShowPassword((v) => !v)}
                    className="rounded p-1 hover:text-foreground"
                  >
                    {showPassword ? <EyeOff aria-hidden /> : <Eye aria-hidden />}
                  </button>
                }
              />
            )}
          </Field>
        </DialogBody>
        <DialogFooter>
          <Button variant="ghost" onClick={onClose} disabled={busy}>
            Cancel
          </Button>
          <Button onClick={() => void save()} disabled={busy || !email || !username}>
            Create
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
