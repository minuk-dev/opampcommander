'use client';

import { RefreshCcw } from 'lucide-react';
import { useCallback, useState } from 'react';
import { Button, useToast } from '@shared/ui';
import { reconcileResource, type ReconcileKind } from '../api/reconcile';

interface ReconcileButtonProps {
  kind: ReconcileKind;
  namespace: string;
  /** Resource name, or instance UID when kind is 'agent'. */
  name: string;
  label?: string;
  variant?: 'default' | 'outline' | 'ghost' | 'secondary';
  /** Called after a successful reconcile, e.g. to refresh the view. */
  onReconciled?: () => void;
}

// ReconcileButton triggers an on-demand reconcile of a single resource and
// reports the outcome through the app-wide toast, so it can be dropped into any
// detail page or list without the host wiring up feedback.
export default function ReconcileButton({
  kind,
  namespace,
  name,
  label = 'Reconcile',
  variant = 'outline',
  onReconciled,
}: ReconcileButtonProps) {
  const [busy, setBusy] = useState(false);
  const { toast } = useToast();

  const run = useCallback(async () => {
    setBusy(true);
    try {
      await reconcileResource(kind, namespace, name);
      toast('success', `Reconciled ${kind} "${name}".`);
      onReconciled?.();
    } catch (err) {
      toast('error', err instanceof Error ? err.message : `Failed to reconcile ${kind}.`);
    } finally {
      setBusy(false);
    }
  }, [kind, namespace, name, onReconciled, toast]);

  return (
    <Button variant={variant} size="sm" onClick={() => void run()} disabled={busy}>
      <RefreshCcw aria-hidden />
      {label}
    </Button>
  );
}
