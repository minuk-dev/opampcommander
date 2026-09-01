'use client';

import { useState } from 'react';
import { RefreshCw } from 'lucide-react';
import Link from 'next/link';
import {
  Alert,
  Badge,
  Button,
  Field,
  PageHeader,
  PaginationFooter,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Spinner,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeaderCell,
  TableRow,
  TableWrap,
} from '@shared/ui';
import { useNamespace } from '@entities/namespace';
import { TimeDisplay } from '@shared/preferences';
import { cn, useCursorPagination } from '@shared/lib';
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
    <div>
      <PageHeader
        title="Connections"
        subtitle={`Namespace: ${namespace}`}
        actions={
          <Button variant="ghost" size="icon-sm" aria-label="Refresh" onClick={() => refresh()}>
            <RefreshCw className={cn(loading && 'animate-spin')} aria-hidden />
          </Button>
        }
      />

      <Field label="Server" className="mb-3 max-w-60">
        {(field) => (
          <Select value={selected} onValueChange={setSelected}>
            <SelectTrigger {...field}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL_SERVERS}>All servers</SelectItem>
              <SelectItem value={LOCAL_NODE}>This node</SelectItem>
              {servers.map((s) => (
                <SelectItem key={s.id} value={s.id} className="font-mono text-xs">
                  {s.id}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        )}
      </Field>

      {error && (
        <Alert severity="error" className="mb-3">
          {error}
        </Alert>
      )}
      <TableWrap>
        <Table>
          <TableHead>
            <TableRow className="hover:bg-transparent">
              <TableHeaderCell>Connection ID</TableHeaderCell>
              <TableHeaderCell>Instance UID</TableHeaderCell>
              {isCluster && <TableHeaderCell>Server</TableHeaderCell>}
              <TableHeaderCell>Type</TableHeaderCell>
              <TableHeaderCell>Alive</TableHeaderCell>
              <TableHeaderCell>Last communicated</TableHeaderCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {loading ? (
              <TableRow className="hover:bg-transparent">
                <TableCell colSpan={columnCount} className="py-8">
                  <Spinner className="mx-auto size-5" />
                </TableCell>
              </TableRow>
            ) : items.length === 0 ? (
              <TableRow className="hover:bg-transparent">
                <TableCell colSpan={columnCount} className="py-8 text-center text-muted-foreground">
                  No connections
                </TableCell>
              </TableRow>
            ) : (
              items.map((c) => (
                <TableRow key={isCluster ? `${c.serverId ?? ''}/${c.id}` : c.id}>
                  <TableCell className="font-mono text-xs">{c.id}</TableCell>
                  <TableCell className="font-mono text-xs">
                    <Link
                      href={`/agents/${c.instanceUid}`}
                      className="text-primary hover:underline"
                    >
                      {c.instanceUid}
                    </Link>
                  </TableCell>
                  {isCluster && (
                    <TableCell className="font-mono text-xs">{c.serverId || '—'}</TableCell>
                  )}
                  <TableCell>{c.type}</TableCell>
                  <TableCell>
                    <Badge variant={c.alive ? 'success' : 'muted'}>
                      {c.alive ? 'Alive' : 'Dead'}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <TimeDisplay value={c.lastCommunicatedAt} />
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableWrap>

      <PaginationFooter pagination={pagination} />
    </div>
  );
}
