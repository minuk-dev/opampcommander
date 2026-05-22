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
  ToggleButton,
  ToggleButtonGroup,
  Typography,
} from '@mui/material';
import { useEffect, useState } from 'react';
import { useNamespace } from '@/components/NamespaceProvider';
import { api } from '@/lib/api-client';
import { fromYAML, toYAML } from '@/lib/yaml';
import type { AgentGroup, AgentGroupSpec } from '@/lib/types';

type Format = 'yaml' | 'json';

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

function serialize(value: unknown, format: Format): string {
  if (format === 'yaml') return toYAML(value);
  return JSON.stringify(value ?? {}, null, 2);
}
function parse(text: string, format: Format): unknown {
  const t = text.trim();
  if (!t) return {};
  if (format === 'yaml') return fromYAML(text);
  return JSON.parse(text);
}

export default function AgentGroupEditDialog({ open, mode, initial, onClose, onSaved }: Props) {
  const { namespace } = useNamespace();
  const [format, setFormat] = useState<Format>('yaml');
  const [name, setName] = useState('');
  const [specText, setSpecText] = useState('');
  const [attributesText, setAttributesText] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (open) {
      setError(null);
      setFormat('yaml');
      setName(initial?.metadata.name ?? '');
      setSpecText(serialize(initial?.spec ?? defaultSpec(), 'yaml'));
      setAttributesText(serialize(initial?.metadata.attributes ?? {}, 'yaml'));
    }
  }, [open, initial]);

  const switchFormat = (next: Format) => {
    if (next === format) return;
    try {
      const spec = parse(specText, format);
      const attrs = parse(attributesText, format);
      setSpecText(serialize(spec, next));
      setAttributesText(serialize(attrs, next));
      setFormat(next);
      setError(null);
    } catch (err) {
      setError(
        `Cannot switch to ${next.toUpperCase()} — current ${format.toUpperCase()} buffer is invalid: ${
          err instanceof Error ? err.message : String(err)
        }`,
      );
    }
  };

  const save = async () => {
    setBusy(true);
    setError(null);
    try {
      const spec = parse(specText, format) as AgentGroupSpec;
      const attributes = parse(attributesText, format) as Record<string, string>;
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
        await api.put(`/api/v1/namespaces/${namespace}/agentgroups/${initial.metadata.name}`, body);
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
      <DialogTitle>
        <Stack direction="row" alignItems="center" justifyContent="space-between">
          {mode === 'create' ? 'Create agent group' : 'Edit agent group'}
          <ToggleButtonGroup
            size="small"
            exclusive
            value={format}
            onChange={(_, v: Format | null) => v && switchFormat(v)}
            aria-label="format"
          >
            <ToggleButton value="yaml">YAML</ToggleButton>
            <ToggleButton value="json">JSON</ToggleButton>
          </ToggleButtonGroup>
        </Stack>
      </DialogTitle>
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
            Attributes ({format.toUpperCase()}, key/value pairs)
          </Typography>
          <TextField
            value={attributesText}
            onChange={(e) => setAttributesText(e.target.value)}
            multiline
            minRows={3}
            spellCheck={false}
            slotProps={{
              input: {
                sx: { fontFamily: 'var(--font-geist-mono), monospace', fontSize: 13 },
              },
            }}
          />
          <Typography variant="body2" color="text.secondary">
            Spec ({format.toUpperCase()}) — <code>priority</code>, <code>selector</code>,{' '}
            <code>agentConfig</code>.
          </Typography>
          <TextField
            value={specText}
            onChange={(e) => setSpecText(e.target.value)}
            multiline
            minRows={14}
            spellCheck={false}
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
        <Button variant="contained" onClick={save} disabled={busy || (mode === 'create' && !name)}>
          Save
        </Button>
      </DialogActions>
    </Dialog>
  );
}
