'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useNamespace, type Namespace } from '@entities/namespace';
import { api } from '@shared/api';
import {
  Button,
  Dialog,
  DialogBody,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Field,
  Input,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@shared/ui';

// Sentinels that turn menu entries into actions rather than selections.
const CREATE = '__create__';
const MANAGE = '__manage__';

export default function NamespaceSelector() {
  const router = useRouter();
  const { namespace, setNamespace, namespaces, refresh } = useNamespace();
  const [createOpen, setCreateOpen] = useState(false);
  const [newName, setNewName] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleChange = (val: string) => {
    if (val === CREATE) return setCreateOpen(true);
    if (val === MANAGE) return router.push('/namespaces');
    setNamespace(val);
  };

  const create = async () => {
    if (!newName) return;
    setBusy(true);
    setError(null);
    try {
      await api.post<Namespace>('/api/v1/namespaces', { metadata: { name: newName } });
      await refresh();
      setNamespace(newName);
      setCreateOpen(false);
      setNewName('');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'failed to create namespace');
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <Select value={namespace} onValueChange={handleChange}>
        <SelectTrigger className="w-36 sm:w-56" aria-label="Namespace">
          {/* Group the prefix with the value so justify-between only pushes the
              chevron to the far edge. */}
          <span className="flex min-w-0 items-center gap-1.5">
            <span className="hidden shrink-0 text-xs text-muted-foreground sm:inline">ns</span>
            <SelectValue />
          </span>
        </SelectTrigger>
        <SelectContent>
          {namespaces.length === 0 && <SelectItem value={namespace}>{namespace}</SelectItem>}
          {namespaces.map((n) => (
            <SelectItem key={n.metadata.name} value={n.metadata.name}>
              {n.metadata.name}
            </SelectItem>
          ))}
          <div className="my-1 h-px bg-border" />
          <SelectItem value={CREATE}>+ Create namespace…</SelectItem>
          <SelectItem value={MANAGE}>Manage namespaces…</SelectItem>
        </SelectContent>
      </Select>

      <Dialog open={createOpen} onOpenChange={(next) => !next && setCreateOpen(false)}>
        <DialogContent size="sm" className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>Create namespace</DialogTitle>
          </DialogHeader>
          <DialogBody>
            <Field label="Name" error={error}>
              {(field) => (
                <Input
                  {...field}
                  autoFocus
                  value={newName}
                  invalid={Boolean(error)}
                  onChange={(e) => setNewName(e.target.value)}
                />
              )}
            </Field>
          </DialogBody>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setCreateOpen(false)} disabled={busy}>
              Cancel
            </Button>
            <Button onClick={() => void create()} disabled={busy || !newName}>
              Create
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
