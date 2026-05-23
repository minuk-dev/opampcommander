'use client';

import {
  Alert,
  Box,
  Chip,
  CircularProgress,
  IconButton,
  Paper,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
} from '@mui/material';
import { Refresh as RefreshIcon } from '@mui/icons-material';
import { useCallback, useEffect, useState } from 'react';
import PageHeader from '@/components/PageHeader';
import { api } from '@/lib/api-client';
import type { ListResponse, Server } from '@/lib/types';

export default function ServersPage() {
  const [items, setItems] = useState<Server[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchItems = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.get<ListResponse<Server>>('/api/v1/servers');
      setItems(res.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void fetchItems();
  }, [fetchItems]);

  return (
    <Box>
      <PageHeader
        title="Servers"
        subtitle="API server cluster members"
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
              <TableCell>Server ID</TableCell>
              <TableCell>Last heartbeat</TableCell>
              <TableCell>Conditions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {loading ? (
              <TableRow>
                <TableCell colSpan={3} align="center">
                  <CircularProgress size={24} />
                </TableCell>
              </TableRow>
            ) : items.length === 0 ? (
              <TableRow>
                <TableCell colSpan={3} align="center">
                  No servers
                </TableCell>
              </TableRow>
            ) : (
              items.map((s) => (
                <TableRow key={s.id} hover>
                  <TableCell sx={{ fontFamily: 'monospace' }}>{s.id}</TableCell>
                  <TableCell>{s.lastHeartbeatAt}</TableCell>
                  <TableCell>
                    <Stack direction="row" gap={0.5} flexWrap="wrap">
                      {(s.conditions ?? []).map((c, i) => (
                        <Chip
                          key={`${c.type}-${i}`}
                          label={`${c.type}: ${c.status}`}
                          color={
                            c.status === 'True'
                              ? 'success'
                              : c.status === 'False'
                                ? 'warning'
                                : 'default'
                          }
                          size="small"
                          variant="outlined"
                        />
                      ))}
                    </Stack>
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
