'use client';

import {
  Alert,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  ListItemText,
  Menu,
  MenuItem,
  Stack,
  TextField,
  ToggleButton,
  ToggleButtonGroup,
  Typography,
} from '@mui/material';
import { ArrowDropDown as ArrowDropDownIcon } from '@mui/icons-material';
import { useEffect, useRef, useState } from 'react';
import { useNamespace } from '@entities/namespace';
import { api } from '@shared/api';
import { loadAgentGroupSamples, type AgentGroupSample } from '../model/samples';
import { fromYAML, toYAML } from '@shared/lib';
import type { AgentGroup, AgentGroupSpec } from '@entities/agent-group';

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
  const [sampleAnchor, setSampleAnchor] = useState<HTMLElement | null>(null);
  const [samples, setSamples] = useState<AgentGroupSample[] | null>(null);
  const [samplesError, setSamplesError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) {
      setSamples(null);
      setSamplesError(null);
      return;
    }
    let cancelled = false;
    loadAgentGroupSamples()
      .then((list) => {
        if (!cancelled) setSamples(list);
      })
      .catch((err) => {
        if (!cancelled) {
          setSamplesError(err instanceof Error ? err.message : String(err));
          setSamples([]);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [open]);

  const applySample = (s: AgentGroupSample) => {
    if (mode === 'create') {
      setName(s.name);
    }
    setAttributesText(serialize(s.attributes, format));
    setSpecText(serialize(s.spec, format));
    setError(null);
    setSampleAnchor(null);
  };

  // Reset on the closed→open transition only. Parents may pass a freshly
  // fetched `initial` reference for the same logical row mid-edit (e.g. list
  // refresh), which must not stomp the user's in-progress buffers.
  const wasOpen = useRef(false);
  const initialRef = useRef(initial);
  initialRef.current = initial;
  useEffect(() => {
    if (open && !wasOpen.current) {
      const i = initialRef.current;
      setError(null);
      setFormat('yaml');
      setName(i?.metadata.name ?? '');
      setSpecText(serialize(i?.spec ?? defaultSpec(), 'yaml'));
      setAttributesText(serialize(i?.metadata.attributes ?? {}, 'yaml'));
    }
    wasOpen.current = open;
  }, [open]);

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
      const parsedSpec = parse(specText, format);
      const parsedAttrs = parse(attributesText, format);
      if (!parsedSpec || typeof parsedSpec !== 'object' || Array.isArray(parsedSpec)) {
        throw new Error('spec must be an object');
      }
      if (!parsedAttrs || typeof parsedAttrs !== 'object' || Array.isArray(parsedAttrs)) {
        throw new Error('attributes must be an object');
      }
      const spec = parsedSpec as AgentGroupSpec;
      const attributes = parsedAttrs as Record<string, string>;
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
        <Stack direction="row" alignItems="center" justifyContent="space-between" gap={1}>
          {mode === 'create' ? 'Create agent group' : 'Edit agent group'}
          <Stack direction="row" alignItems="center" gap={1}>
            <Button
              size="small"
              variant="outlined"
              endIcon={<ArrowDropDownIcon />}
              onClick={(e) => setSampleAnchor(e.currentTarget)}
              aria-label="load sample"
              disabled={!samples}
            >
              {samples ? 'Load sample' : 'Loading…'}
            </Button>
            <Menu
              anchorEl={sampleAnchor}
              open={Boolean(sampleAnchor)}
              onClose={() => setSampleAnchor(null)}
            >
              {(samples ?? []).length === 0 && <MenuItem disabled>No samples available</MenuItem>}
              {(samples ?? []).map((s, i) => (
                <MenuItem key={`${i}-${s.label}`} onClick={() => applySample(s)}>
                  <ListItemText primary={s.label} secondary={s.description} />
                </MenuItem>
              ))}
            </Menu>
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
        </Stack>
      </DialogTitle>
      <DialogContent>
        <Stack spacing={2} mt={1}>
          {samplesError && <Alert severity="warning">Failed to load samples: {samplesError}</Alert>}
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
