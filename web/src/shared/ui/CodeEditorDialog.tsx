'use client';

import {
  Alert,
  Box,
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
import { type ReactNode, useEffect, useRef, useState } from 'react';
import { loadSamples, type SamplesPath, fromYAML, toYAML } from '@shared/lib';

export type CodeFormat = 'yaml' | 'json';

export interface CodeSample {
  label: string;
  description?: string;
  value: unknown;
}

interface Props {
  open: boolean;
  title: string;
  description?: ReactNode;
  initialValue: unknown;
  defaultFormat?: CodeFormat;
  // Inline list of samples (synchronous). Prefer samplesUrl in new code.
  samples?: CodeSample[];
  // Path under /samples/*.yaml (see web/public/samples/). When set, the menu
  // is populated on dialog open from this file. {{namespace}}, {{now}}, etc.
  // in the YAML are substituted from samplesVars (plus a built-in `now`).
  samplesUrl?: SamplesPath;
  samplesVars?: Record<string, string>;
  onClose: () => void;
  onSave: (parsed: unknown) => Promise<void> | void;
}

function serialize(value: unknown, format: CodeFormat): string {
  if (format === 'yaml') return toYAML(value);
  return JSON.stringify(value ?? {}, null, 2);
}

function parse(text: string, format: CodeFormat): unknown {
  const trimmed = text.trim();
  if (trimmed === '') return {};
  if (format === 'yaml') return fromYAML(text);
  return JSON.parse(text);
}

// CodeEditorDialog is the canonical editor for structured payloads. YAML is
// the default surface; users can flip to JSON, and the current buffer is
// re-serialized in the new format so they don't lose work.
export default function CodeEditorDialog({
  open,
  title,
  description,
  initialValue,
  defaultFormat = 'yaml',
  samples,
  samplesUrl,
  samplesVars,
  onClose,
  onSave,
}: Props) {
  const [format, setFormat] = useState<CodeFormat>(defaultFormat);
  const [text, setText] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [sampleAnchor, setSampleAnchor] = useState<HTMLElement | null>(null);
  const [loadedSamples, setLoadedSamples] = useState<CodeSample[] | null>(null);
  const [samplesError, setSamplesError] = useState<string | null>(null);

  // Reset buffer only on the closed→open transition. Parents commonly pass a
  // freshly-constructed initialValue (e.g. emptyFoo()) each render; depending
  // on its identity would wipe in-progress edits on every parent re-render.
  const wasOpen = useRef(false);
  const initialValueRef = useRef(initialValue);
  const defaultFormatRef = useRef(defaultFormat);
  initialValueRef.current = initialValue;
  defaultFormatRef.current = defaultFormat;
  useEffect(() => {
    if (open && !wasOpen.current) {
      setFormat(defaultFormatRef.current);
      setText(serialize(initialValueRef.current, defaultFormatRef.current));
      setError(null);
    }
    wasOpen.current = open;
  }, [open]);

  // Stable JSON key so we don't refetch every render when the parent creates
  // a fresh samplesVars object each time. samplesVars itself MUST stay out of
  // the dep array — using both defeats the stabilization.
  const varsKey = samplesVars ? JSON.stringify(samplesVars) : '';
  const samplesVarsRef = useRef(samplesVars);
  samplesVarsRef.current = samplesVars;
  useEffect(() => {
    if (!open) {
      // Drop stale samples loaded for a previous open/URL so the next open
      // shows "Loading…" instead of the previous file's entries.
      setLoadedSamples(null);
      setSamplesError(null);
      return;
    }
    if (!samplesUrl) return;
    setLoadedSamples(null);
    setSamplesError(null);
    let cancelled = false;
    loadSamples(samplesUrl, samplesVarsRef.current ?? {})
      .then((list) => {
        if (!cancelled) setLoadedSamples(list);
      })
      .catch((err) => {
        if (!cancelled) {
          setSamplesError(err instanceof Error ? err.message : String(err));
          setLoadedSamples([]);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [open, samplesUrl, varsKey]);

  const effectiveSamples = samples ?? loadedSamples ?? [];

  const switchFormat = (next: CodeFormat) => {
    if (next === format) return;
    // Re-serialize the current buffer so unsaved edits survive the toggle.
    try {
      const parsed = parse(text, format);
      setText(serialize(parsed, next));
      setError(null);
      setFormat(next);
    } catch (err) {
      setError(
        `Cannot switch to ${next.toUpperCase()} — current ${format.toUpperCase()} buffer is invalid: ${
          err instanceof Error ? err.message : String(err)
        }`,
      );
    }
  };

  const applySample = (sample: CodeSample) => {
    setText(serialize(sample.value, format));
    setError(null);
    setSampleAnchor(null);
  };

  const save = async () => {
    setBusy(true);
    setError(null);
    try {
      const parsed = parse(text, format);
      await onSave(parsed);
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
          <Box>{title}</Box>
          <Stack direction="row" alignItems="center" gap={1}>
            {(samples || samplesUrl) && (
              <>
                <Button
                  size="small"
                  variant="outlined"
                  endIcon={<ArrowDropDownIcon />}
                  onClick={(e) => setSampleAnchor(e.currentTarget)}
                  aria-label="load sample"
                  disabled={!samples && !loadedSamples}
                >
                  {samples || loadedSamples ? 'Load sample' : 'Loading…'}
                </Button>
                <Menu
                  anchorEl={sampleAnchor}
                  open={Boolean(sampleAnchor)}
                  onClose={() => setSampleAnchor(null)}
                >
                  {effectiveSamples.length === 0 && (
                    <MenuItem disabled>No samples available</MenuItem>
                  )}
                  {effectiveSamples.map((s, i) => (
                    <MenuItem key={`${i}-${s.label}`} onClick={() => applySample(s)}>
                      <ListItemText primary={s.label} secondary={s.description} />
                    </MenuItem>
                  ))}
                </Menu>
              </>
            )}
            <ToggleButtonGroup
              size="small"
              exclusive
              value={format}
              onChange={(_, v: CodeFormat | null) => v && switchFormat(v)}
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
          {description && (
            <Typography variant="body2" color="text.secondary">
              {description}
            </Typography>
          )}
          {samplesError && <Alert severity="warning">Failed to load samples: {samplesError}</Alert>}
          {error && <Alert severity="error">{error}</Alert>}
          <TextField
            value={text}
            onChange={(e) => setText(e.target.value)}
            multiline
            minRows={18}
            spellCheck={false}
            slotProps={{
              input: {
                sx: {
                  fontFamily: 'var(--font-geist-mono), monospace',
                  fontSize: 13,
                },
              },
            }}
          />
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
