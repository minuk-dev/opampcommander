'use client';

import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  InputLabel,
  MenuItem,
  Select,
  TextField,
} from '@mui/material';
import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Divider } from '@mui/material';
import { useNamespace, type Namespace } from '@entities/namespace';
import { api } from '@shared/api';

export default function NamespaceSelector() {
  const router = useRouter();
  const { namespace, setNamespace, namespaces, refresh } = useNamespace();
  const [createOpen, setCreateOpen] = useState(false);
  const [newName, setNewName] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleChange = (val: string) => {
    if (val === '__create__') {
      setCreateOpen(true);
      return;
    }
    if (val === '__manage__') {
      router.push('/namespaces');
      return;
    }
    setNamespace(val);
  };

  const create = async () => {
    if (!newName) return;
    setBusy(true);
    setError(null);
    try {
      await api.post<Namespace>('/api/v1/namespaces', {
        metadata: { name: newName },
      });
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
      <FormControl size="small" sx={{ minWidth: { xs: 120, sm: 200 } }}>
        <InputLabel
          id="namespace-label"
          sx={{ color: 'inherit', '&.Mui-focused': { color: 'inherit' } }}
        >
          Namespace
        </InputLabel>
        <Select
          labelId="namespace-label"
          label="Namespace"
          value={namespace}
          onChange={(e) => handleChange(e.target.value)}
          sx={{
            color: 'inherit',
            '.MuiOutlinedInput-notchedOutline': {
              borderColor: 'rgba(255,255,255,0.4)',
            },
            '&:hover .MuiOutlinedInput-notchedOutline': {
              borderColor: 'rgba(255,255,255,0.7)',
            },
            '.MuiSvgIcon-root': { color: 'inherit' },
          }}
        >
          {namespaces.length === 0 && <MenuItem value={namespace}>{namespace}</MenuItem>}
          {namespaces.map((n) => (
            <MenuItem key={n.metadata.name} value={n.metadata.name}>
              {n.metadata.name}
            </MenuItem>
          ))}
          <Divider />
          <MenuItem value="__create__">+ Create namespace…</MenuItem>
          <MenuItem value="__manage__">Manage namespaces…</MenuItem>
        </Select>
      </FormControl>

      <Dialog open={createOpen} onClose={() => setCreateOpen(false)} fullWidth maxWidth="xs">
        <DialogTitle>Create namespace</DialogTitle>
        <DialogContent>
          <Box pt={1}>
            <TextField
              autoFocus
              fullWidth
              label="Name"
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              error={Boolean(error)}
              helperText={error}
            />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCreateOpen(false)} disabled={busy}>
            Cancel
          </Button>
          <Button onClick={create} variant="contained" disabled={busy || !newName}>
            Create
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}
