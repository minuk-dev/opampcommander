'use client';

import { useCallback, useState } from 'react';
import { Alert, Button, Snackbar, type ButtonProps } from '@mui/material';
import { Sync as SyncIcon } from '@mui/icons-material';
import { reconcileResource, type ReconcileKind } from '../api/reconcile';

interface ReconcileButtonProps {
  kind: ReconcileKind;
  namespace: string;
  /** Resource name, or instance UID when kind is 'agent'. */
  name: string;
  label?: string;
  variant?: ButtonProps['variant'];
  /** Called after a successful reconcile, e.g. to refresh the view. */
  onReconciled?: () => void;
}

type Feedback = { severity: 'success' | 'error'; message: string };

// ReconcileButton triggers an on-demand reconcile of a single resource and reports the
// outcome via a self-contained snackbar, so it can be dropped into any detail page or list
// without the host wiring up feedback.
export default function ReconcileButton({
  kind,
  namespace,
  name,
  label = 'Reconcile',
  variant = 'outlined',
  onReconciled,
}: ReconcileButtonProps) {
  const [busy, setBusy] = useState(false);
  const [feedback, setFeedback] = useState<Feedback | null>(null);

  const run = useCallback(async () => {
    setBusy(true);
    try {
      await reconcileResource(kind, namespace, name);
      setFeedback({ severity: 'success', message: `Reconciled ${kind} "${name}".` });
      onReconciled?.();
    } catch (err) {
      setFeedback({
        severity: 'error',
        message: err instanceof Error ? err.message : `Failed to reconcile ${kind}.`,
      });
    } finally {
      setBusy(false);
    }
  }, [kind, namespace, name, onReconciled]);

  return (
    <>
      <Button startIcon={<SyncIcon />} variant={variant} onClick={run} disabled={busy}>
        {label}
      </Button>
      <Snackbar
        open={feedback !== null}
        autoHideDuration={4000}
        onClose={() => setFeedback(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
      >
        {feedback === null ? undefined : (
          <Alert severity={feedback.severity} onClose={() => setFeedback(null)} variant="filled">
            {feedback.message}
          </Alert>
        )}
      </Snackbar>
    </>
  );
}
