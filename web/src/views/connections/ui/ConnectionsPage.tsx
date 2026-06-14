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
import { PageHeader, PaginationFooter } from '@shared/ui';
import { useNamespace } from '@entities/namespace';
import { TimeDisplay } from '@shared/preferences';
import { useCursorPagination } from '@shared/lib';
import type { Connection } from '@entities/connection';

export default function ConnectionsPage() {
  const { namespace } = useNamespace();
  const pagination = useCursorPagination<Connection>(`/api/v1/namespaces/${namespace}/connections`);
  const { items, isLoading: loading, error: fetchError, refresh } = pagination;
  const error =
    fetchError instanceof Error ? fetchError.message : fetchError ? 'Failed to fetch' : null;

  return (
    <Box>
      <PageHeader
        title="Connections"
        subtitle={`Namespace: ${namespace}`}
        actions={
          <IconButton color="primary" onClick={() => refresh()}>
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

      <PaginationFooter pagination={pagination} />
    </Box>
  );
}
