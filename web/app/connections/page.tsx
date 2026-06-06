'use client';

import {
  Alert,
  Box,
  Chip,
  CircularProgress,
  IconButton,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
} from '@mui/material';
import { Refresh as RefreshIcon } from '@mui/icons-material';
import Link from 'next/link';
import { useCallback, useEffect, useState } from 'react';
import PageHeader from '@/components/PageHeader';
import { useNamespace } from '@/components/NamespaceProvider';
import TimeDisplay from '@/components/TimeDisplay';
import { api } from '@/lib/api-client';
import type { Connection, ListResponse } from '@/lib/types';

export default function ConnectionsPage() {
  const { namespace } = useNamespace();
  const [items, setItems] = useState<Connection[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchItems = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.get<ListResponse<Connection>>(
        `/api/v1/namespaces/${namespace}/connections`,
        { query: { limit: 200 } },
      );
      setItems(res.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch');
    } finally {
      setLoading(false);
    }
  }, [namespace]);

  useEffect(() => {
    void fetchItems();
  }, [fetchItems]);

  return (
    <Box>
      <PageHeader
        title="Connections"
        subtitle={`Namespace: ${namespace}`}
        actions={
          <IconButton color="primary" onClick={fetchItems}>
            <RefreshIcon />
          </IconButton>
        }
      />
      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}
      <TableContainer component={Paper}>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>Connection ID</TableCell>
              <TableCell>Instance UID</TableCell>
              <TableCell>Type</TableCell>
              <TableCell>Alive</TableCell>
              <TableCell>Last communicated</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {loading ? (
              <TableRow>
                <TableCell colSpan={5} align="center">
                  <CircularProgress size={24} />
                </TableCell>
              </TableRow>
            ) : items.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} align="center">
                  No connections
                </TableCell>
              </TableRow>
            ) : (
              items.map((c) => (
                <TableRow key={c.id} hover>
                  <TableCell sx={{ fontFamily: 'monospace' }}>{c.id}</TableCell>
                  <TableCell sx={{ fontFamily: 'monospace' }}>
                    <Link href={`/agents/${c.instanceUid}`}>{c.instanceUid}</Link>
                  </TableCell>
                  <TableCell>{c.type}</TableCell>
                  <TableCell>
                    <Chip
                      label={c.alive ? 'Alive' : 'Dead'}
                      color={c.alive ? 'success' : 'default'}
                      size="small"
                    />
                  </TableCell>
                  <TableCell>
                    <TimeDisplay value={c.lastCommunicatedAt} />
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>
    </Box>
  );
}
