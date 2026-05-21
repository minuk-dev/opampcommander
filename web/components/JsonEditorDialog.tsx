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
import { ReactNode, useEffect, useState } from 'react';

interface Props {
  open: boolean;
  title: string;
  description?: ReactNode;
  initialValue: unknown;
  onClose: () => void;
  onSave: (parsed: unknown) => Promise<void> | void;
}

export default function JsonEditorDialog({
  open,
  title,
  description,
  initialValue,
  onClose,
  onSave,
}: Props) {
  const [text, setText] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (open) {
      setText(JSON.stringify(initialValue ?? {}, null, 2));
      setError(null);
    }
  }, [open, initialValue]);

  const save = async () => {
    setBusy(true);
    setError(null);
    try {
      const parsed = text.trim() ? JSON.parse(text) : {};
      await onSave(parsed);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save');
    } finally {
      setBusy(false);
    }
  };

  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="md">
      <DialogTitle>{title}</DialogTitle>
      <DialogContent>
        <Stack spacing={2} mt={1}>
          {description && (
            <Typography variant="body2" color="text.secondary">{description}</Typography>
          )}
          {error && <Alert severity="error">{error}</Alert>}
          <TextField
            value={text}
            onChange={(e) => setText(e.target.value)}
            multiline
            minRows={18}
            slotProps={{
              input: { sx: { fontFamily: 'var(--font-geist-mono), monospace', fontSize: 13 } },
            }}
          />
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={busy}>Cancel</Button>
        <Button variant="contained" onClick={save} disabled={busy}>Save</Button>
      </DialogActions>
    </Dialog>
  );
}
