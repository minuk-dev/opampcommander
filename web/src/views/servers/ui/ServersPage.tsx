'use client';

import { RefreshCw } from 'lucide-react';
import { useCallback, useEffect, useState } from 'react';
import { api, type ListResponse } from '@shared/api';
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
    <div>
      <PageHeader
        title="Servers"
        subtitle="API server cluster members"
        actions={
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label="Refresh"
            onClick={() => void fetchItems()}
          >
            <RefreshCw className={cn(loading && 'animate-spin')} aria-hidden />
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
            {loading ? (
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
