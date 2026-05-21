'use client';

import {
  Alert,
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
import type { AgentGroup, AgentGroupSpec } from '@/lib/types';

interface Props {
  open: boolean;
  mode: 'create' | 'edit';
  initial?: AgentGroup;
  onClose: () => void;
  onSaved: () => void;
}

function defaultSpec(): AgentGroupSpec {
  return {
    priority: 0,
    selector: {
      identifyingAttributes: {},
      nonIdentifyingAttributes: {},
    },
  };
}

export default function AgentGroupEditDialog({
  open,
  mode,
  initial,
  onClose,
  onSaved,
}: Props) {
  const { namespace } = useNamespace();
  const [name, setName] = useState('');
  const [specText, setSpecText] = useState('');
  const [attributesText, setAttributesText] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (open) {
      setError(null);
      setName(initial?.metadata.name ?? '');
      setSpecText(JSON.stringify(initial?.spec ?? defaultSpec(), null, 2));
      setAttributesText(
        JSON.stringify(initial?.metadata.attributes ?? {}, null, 2),
      );
    }
  }, [open, initial]);

  const save = async () => {
    setBusy(true);
    setError(null);
    try {
      const spec = JSON.parse(specText) as AgentGroupSpec;
      const attributes = attributesText.trim()
        ? (JSON.parse(attributesText) as Record<string, string>)
        : {};
      if (mode === 'create') {
        const body: Partial<AgentGroup> = {
          metadata: {
            namespace,
            name,
            attributes,
            createdAt: new Date().toISOString(),
          },
          spec,
        };
        await api.post(`/api/v1/namespaces/${namespace}/agentgroups`, body);
      } else if (initial) {
        const body: AgentGroup = {
          ...initial,
          metadata: { ...initial.metadata, attributes },
          spec,
        };
        await api.put(
          `/api/v1/namespaces/${namespace}/agentgroups/${initial.metadata.name}`,
          body,
        );
      }
      onSaved();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save');
    } finally {
      setBusy(false);
    }
  };

  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="md">
      <DialogTitle>{mode === 'create' ? 'Create agent group' : 'Edit agent group'}</DialogTitle>
      <DialogContent>
        <Stack spacing={2} mt={1}>
          {error && <Alert severity="error">{error}</Alert>}
          <TextField
            label="Name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            disabled={mode === 'edit'}
            fullWidth
            required
          />
          <Typography variant="body2" color="text.secondary">
            Attributes (JSON, key/value pairs)
          </Typography>
          <TextField
            value={attributesText}
            onChange={(e) => setAttributesText(e.target.value)}
            multiline
            minRows={3}
            slotProps={{
              input: {
                sx: { fontFamily: 'var(--font-geist-mono), monospace', fontSize: 13 },
              },
            }}
          />
          <Typography variant="body2" color="text.secondary">
            Spec (JSON) — <code>priority</code>, <code>selector</code>,{' '}
            <code>agentConfig</code>.
          </Typography>
          <TextField
            value={specText}
            onChange={(e) => setSpecText(e.target.value)}
            multiline
            minRows={14}
            slotProps={{
              input: {
                sx: { fontFamily: 'var(--font-geist-mono), monospace', fontSize: 13 },
              },
            }}
          />
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={busy}>
          Cancel
        </Button>
        <Button
          variant="contained"
          onClick={save}
          disabled={busy || (mode === 'create' && !name)}
        >
          Save
        </Button>
      </DialogActions>
    </Dialog>
  );
}
