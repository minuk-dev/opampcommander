'use client';

import { useState } from 'react';
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
  ToggleButton,
  ToggleButtonGroup,
} from '@mui/material';
import { Refresh as RefreshIcon } from '@mui/icons-material';
import Link from 'next/link';
import { PageHeader, PaginationFooter } from '@shared/ui';
import { useNamespace } from '@entities/namespace';
import { TimeDisplay } from '@shared/preferences';
import { useCursorPagination } from '@shared/lib';
import type { Connection } from '@entities/connection';

// Page connections in small batches so a large cluster-wide listing never pulls
// thousands of rows at once.
const PAGE_SIZE = 20;

type Scope = 'cluster' | 'local';

export default function ConnectionsPage() {
  const { namespace } = useNamespace();
  const [scope, setScope] = useState<Scope>('cluster');
  const isCluster = scope === 'cluster';

  const pagination = useCursorPagination<Connection>(
    `/api/v1/namespaces/${namespace}/connections`,
    {
      initialPageSize: PAGE_SIZE,
      // Cluster scope aggregates every server's connections (each carries its
      // owning serverId); local scope returns only this node's connections.
      query: isCluster ? { scope: 'cluster' } : undefined,
    },
  );
  const { items, isLoading: loading, error: fetchError, refresh } = pagination;
  const error =
    fetchError instanceof Error ? fetchError.message : fetchError ? 'Failed to fetch' : null;

  // Server column is only meaningful in the cluster view.
  const columnCount = isCluster ? 6 : 5;

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

      <ToggleButtonGroup
        value={scope}
        exclusive
        size="small"
        color="primary"
        onChange={(_, value: Scope | null) => {
          if (value !== null) setScope(value);
        }}
        sx={{ mb: 2 }}
      >
        <ToggleButton value="cluster">All servers</ToggleButton>
        <ToggleButton value="local">This node</ToggleButton>
      </ToggleButtonGroup>

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
              {isCluster && <TableCell>Server</TableCell>}
              <TableCell>Type</TableCell>
              <TableCell>Alive</TableCell>
              <TableCell>Last communicated</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {loading ? (
              <TableRow>
                <TableCell colSpan={columnCount} align="center">
                  <CircularProgress size={24} />
                </TableCell>
              </TableRow>
            ) : items.length === 0 ? (
              <TableRow>
                <TableCell colSpan={columnCount} align="center">
                  No connections
                </TableCell>
              </TableRow>
            ) : (
              items.map((c) => (
                <TableRow key={isCluster ? `${c.serverId ?? ''}/${c.id}` : c.id} hover>
                  <TableCell sx={{ fontFamily: 'monospace' }}>{c.id}</TableCell>
                  <TableCell sx={{ fontFamily: 'monospace' }}>
                    <Link href={`/agents/${c.instanceUid}`}>{c.instanceUid}</Link>
                  </TableCell>
                  {isCluster && (
                    <TableCell sx={{ fontFamily: 'monospace' }}>{c.serverId || '—'}</TableCell>
                  )}
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
