'use client';

import { useState } from 'react';
import {
  Alert,
  Box,
  Chip,
  CircularProgress,
  FormControl,
  IconButton,
  InputLabel,
  MenuItem,
  Paper,
  Select,
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
import { useApi, type ListResponse } from '@shared/api';
import type { Connection } from '@entities/connection';
import type { Server } from '@entities/server';

// Page connections in small batches so a large cluster-wide listing never pulls
// thousands of rows at once.
const PAGE_SIZE = 20;

// Sentinel Select values that aren't a concrete server id.
const ALL_SERVERS = '';
const LOCAL_NODE = '__local__';

export default function ConnectionsPage() {
  const { namespace } = useNamespace();
  // Selected server: '' = all servers (cluster), '__local__' = this node only,
  // otherwise a specific server id (cluster, filtered to that server).
  const [selected, setSelected] = useState<string>(ALL_SERVERS);

  const { data: serversData } = useApi<ListResponse<Server>>('/api/v1/servers');
  const servers = serversData?.items ?? [];

  const isLocal = selected === LOCAL_NODE;
  const isCluster = !isLocal;
  const serverId = isCluster && selected !== ALL_SERVERS ? selected : undefined;

  const pagination = useCursorPagination<Connection>(
    `/api/v1/namespaces/${namespace}/connections`,
    {
      initialPageSize: PAGE_SIZE,
      // Cluster scope aggregates servers' connections (each carries its owning
      // serverId); a serverId narrows it to one server. Local scope returns only
      // this node's connections. serverId is filtered server-side so pagination stays
      // correct.
      query: isLocal ? undefined : { scope: 'cluster', serverId },
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

      <FormControl size="small" sx={{ mb: 2, minWidth: { xs: 160, sm: 240 } }}>
        <InputLabel id="connection-server-label">Server</InputLabel>
        <Select
          labelId="connection-server-label"
          label="Server"
          value={selected}
          onChange={(e) => setSelected(e.target.value)}
        >
          <MenuItem value={ALL_SERVERS}>All servers</MenuItem>
          <MenuItem value={LOCAL_NODE}>This node</MenuItem>
          {servers.map((s) => (
            <MenuItem key={s.id} value={s.id} sx={{ fontFamily: 'monospace' }}>
              {s.id}
            </MenuItem>
          ))}
        </Select>
      </FormControl>

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
