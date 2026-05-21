'use client';

import {
  Alert,
  Box,
  Button,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  Paper,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
} from '@mui/material';
import {
  Add as AddIcon,
  Delete as DeleteIcon,
  Refresh as RefreshIcon,
} from '@mui/icons-material';
import { useCallback, useEffect, useState } from 'react';
import PageHeader from '@/components/PageHeader';
import ConfirmDialog from '@/components/ConfirmDialog';
import { useNamespace } from '@/components/NamespaceProvider';
import { api } from '@/lib/api-client';
import type { ListResponse, Namespace } from '@/lib/types';

export default function NamespacesPage() {
  const { namespaces: ctxNamespaces, refresh: refreshCtx } = useNamespace();
  const [items, setItems] = useState<Namespace[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [newName, setNewName] = useState('');
  const [labelsText, setLabelsText] = useState('{}');
  const [deleting, setDeleting] = useState<Namespace | null>(null);

  const fetchItems = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.get<ListResponse<Namespace>>('/api/v1/namespaces', {
        query: { limit: 200 },
      });
      setItems(res.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch namespaces');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (ctxNamespaces.length > 0) {
      setItems(ctxNamespaces);
      setLoading(false);
    } else {
      void fetchItems();
    }
  }, [ctxNamespaces, fetchItems]);

  const onCreate = async () => {
    setError(null);
    try {
      const labels = labelsText.trim() ? JSON.parse(labelsText) : undefined;
      await api.post('/api/v1/namespaces', {
        metadata: { name: newName, labels },
      });
      setCreateOpen(false);
      setNewName('');
      setLabelsText('{}');
      await fetchItems();
      await refreshCtx();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create');
    }
  };

  const onDelete = async () => {
    if (!deleting) return;
    try {
      await api.delete(`/api/v1/namespaces/${deleting.metadata.name}`);
      setDeleting(null);
      await fetchItems();
      await refreshCtx();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete');
    }
  };

  return (
    <Box>
      <PageHeader
        title="Namespaces"
        actions={
          <>
            <IconButton color="primary" onClick={fetchItems}>
              <RefreshIcon />
            </IconButton>
            <Button startIcon={<AddIcon />} variant="contained" onClick={() => setCreateOpen(true)}>
              New namespace
            </Button>
          </>
        }
      />

      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      <TableContainer component={Paper}>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>Name</TableCell>
              <TableCell>Labels</TableCell>
              <TableCell>Created</TableCell>
              <TableCell>Deleted</TableCell>
              <TableCell align="right">Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {loading ? (
              <TableRow><TableCell colSpan={5} align="center"><CircularProgress size={24} /></TableCell></TableRow>
            ) : items.length === 0 ? (
              <TableRow><TableCell colSpan={5} align="center">No namespaces</TableCell></TableRow>
            ) : (
              items.map((ns) => (
                <TableRow key={ns.metadata.name} hover>
                  <TableCell>{ns.metadata.name}</TableCell>
                  <TableCell sx={{ fontFamily: 'monospace', fontSize: 12 }}>
                    {ns.metadata.labels ? JSON.stringify(ns.metadata.labels) : '-'}
                  </TableCell>
                  <TableCell>{ns.metadata.createdAt}</TableCell>
                  <TableCell>{ns.metadata.deletedAt || '-'}</TableCell>
                  <TableCell align="right">
                    <IconButton size="small" onClick={() => setDeleting(ns)}>
                      <DeleteIcon fontSize="small" />
                    </IconButton>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>

      <Dialog open={createOpen} onClose={() => setCreateOpen(false)} fullWidth maxWidth="sm">
        <DialogTitle>Create namespace</DialogTitle>
        <DialogContent>
          <Stack spacing={2} mt={1}>
            <TextField
              label="Name"
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              required
              fullWidth
              autoFocus
            />
            <TextField
              label="Labels (JSON)"
              value={labelsText}
              onChange={(e) => setLabelsText(e.target.value)}
              multiline
              minRows={3}
              slotProps={{
                input: { sx: { fontFamily: 'var(--font-geist-mono), monospace', fontSize: 13 } },
              }}
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCreateOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={onCreate} disabled={!newName}>Create</Button>
        </DialogActions>
      </Dialog>
      <ConfirmDialog
        open={deleting !== null}
        title="Delete namespace"
        message={`Delete "${deleting?.metadata.name}"? This will cascade to its resources.`}
        confirmLabel="Delete"
        destructive
        onClose={() => setDeleting(null)}
        onConfirm={onDelete}
      />
    </Box>
  );
}
