'use client';

import { RefreshCw } from 'lucide-react';
import { useApi, type ListResponse } from '@shared/api';
import { cn } from '@shared/lib';
import { TimeDisplay } from '@shared/preferences';
import {
  Alert,
  Button,
  ConditionBadges,
  PageHeader,
  Spinner,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeaderCell,
  TableRow,
  TableWrap,
} from '@shared/ui';
import type { Server } from '@entities/server';

export default function ServersPage() {
  const {
    data,
    error: fetchError,
    isLoading,
    isValidating,
    mutate,
  } = useApi<ListResponse<Server>>('/api/v1/servers');

  const items = data?.items ?? [];
  const error =
    fetchError instanceof Error ? fetchError.message : fetchError ? 'Failed to fetch' : null;

  return (
    <div>
      <PageHeader
        title="Servers"
        subtitle="API server cluster members"
        actions={
          <Button variant="ghost" size="icon-sm" aria-label="Refresh" onClick={() => void mutate()}>
            <RefreshCw className={cn(isValidating && 'animate-spin')} aria-hidden />
          </Button>
        }
      />
      {error && (
        <Alert severity="error" className="mb-3">
          {error}
        </Alert>
      )}
      <TableWrap>
        <Table>
          <TableHead>
            <TableRow className="hover:bg-transparent">
              <TableHeaderCell>Server ID</TableHeaderCell>
              <TableHeaderCell>Last heartbeat</TableHeaderCell>
              <TableHeaderCell>Conditions</TableHeaderCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {isLoading ? (
              <TableRow className="hover:bg-transparent">
                <TableCell colSpan={3} className="py-8">
                  <Spinner className="mx-auto size-5" />
                </TableCell>
              </TableRow>
            ) : items.length === 0 ? (
              <TableRow className="hover:bg-transparent">
                <TableCell colSpan={3} className="py-8 text-center text-muted-foreground">
                  No servers
                </TableCell>
              </TableRow>
            ) : (
              items.map((s) => (
                <TableRow key={s.id}>
                  <TableCell className="font-mono text-xs">{s.id}</TableCell>
                  <TableCell>
                    <TimeDisplay value={s.lastHeartbeatAt} />
                  </TableCell>
                  <TableCell>
                    <ConditionBadges conditions={s.conditions} />
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableWrap>
    </div>
  );
}
