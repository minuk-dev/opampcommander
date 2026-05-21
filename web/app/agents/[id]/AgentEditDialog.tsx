'use client';

import {
  Alert,
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import { useEffect, useState } from 'react';
import { useNamespace } from '@/components/NamespaceProvider';
import { api } from '@/lib/api-client';
import type { Agent } from '@/lib/types';

interface Props {
  open: boolean;
  agent: Agent;
  onClose: () => void;
  onSaved: (agent: Agent) => void;
}

export default function AgentEditDialog({ open, agent, onClose, onSaved }: Props) {
  const { namespace } = useNamespace();
  const [specText, setSpecText] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (open) {
      setSpecText(JSON.stringify(agent.spec ?? {}, null, 2));
      setError(null);
    }
  }, [open, agent.spec]);

  const save = async () => {
    setBusy(true);
    setError(null);
    try {
      const parsed = specText.trim() ? JSON.parse(specText) : {};
      const next: Agent = { ...agent, spec: parsed };
      const updated = await api.put<Agent>(
        `/api/v1/namespaces/${namespace}/agents/${agent.metadata.instanceUid}`,
        next,
      );
      onSaved(updated);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update agent');
    } finally {
      setBusy(false);
    }
  };

  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="md">
      <DialogTitle>Edit agent spec</DialogTitle>
      <DialogContent>
        <Stack spacing={2}>
          <Typography variant="body2" color="text.secondary">
            Edit the agent&apos;s spec as JSON. Fields:{' '}
            <code>newInstanceUid</code>, <code>connectionSettings</code>,{' '}
            <code>remoteConfig</code>, <code>packagesAvailable</code>,{' '}
            <code>restartRequiredAt</code>.
          </Typography>
          {error && <Alert severity="error">{error}</Alert>}
          <TextField
            label="spec (JSON)"
            multiline
            minRows={16}
            value={specText}
            onChange={(e) => setSpecText(e.target.value)}
            fullWidth
            slotProps={{
              input: {
                sx: { fontFamily: 'var(--font-geist-mono), monospace', fontSize: 13 },
              },
            }}
          />
          <Box>
            <Button
              size="small"
              onClick={() => {
                try {
                  setSpecText(JSON.stringify(JSON.parse(specText), null, 2));
                } catch (err) {
                  setError(err instanceof Error ? err.message : 'invalid JSON');
                }
              }}
            >
              Format
            </Button>
          </Box>
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={busy}>
          Cancel
        </Button>
        <Button variant="contained" onClick={save} disabled={busy}>
          Save
        </Button>
      </DialogActions>
    </Dialog>
  );
}
