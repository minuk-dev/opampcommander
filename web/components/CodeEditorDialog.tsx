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
  ToggleButton,
  ToggleButtonGroup,
  Typography,
} from '@mui/material';
import { type ReactNode, useEffect, useState } from 'react';
import { fromYAML, toYAML } from '@/lib/yaml';

export type CodeFormat = 'yaml' | 'json';

interface Props {
  open: boolean;
  title: string;
  description?: ReactNode;
  initialValue: unknown;
  defaultFormat?: CodeFormat;
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
  onClose,
  onSave,
}: Props) {
  const [format, setFormat] = useState<CodeFormat>(defaultFormat);
  const [text, setText] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (open) {
      setFormat(defaultFormat);
      setText(serialize(initialValue, defaultFormat));
      setError(null);
    }
  }, [open, initialValue, defaultFormat]);

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
        <Stack direction="row" alignItems="center" justifyContent="space-between">
          <Box>{title}</Box>
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
      </DialogTitle>
      <DialogContent>
        <Stack spacing={2} mt={1}>
          {description && (
            <Typography variant="body2" color="text.secondary">
              {description}
            </Typography>
          )}
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
