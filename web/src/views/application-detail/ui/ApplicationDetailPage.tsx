'use client';

import { ArrowLeft, RefreshCw } from 'lucide-react';
import Link from 'next/link';
import { useParams } from 'next/navigation';
import { useApi } from '@shared/api';
import { useCursorPagination } from '@shared/lib';
import { TimeDisplay } from '@shared/preferences';
import {
  Alert,
  Badge,
  Button,
  Card,
  CardContent,
  PageHeader,
  PaginationFooter,
  Spinner,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeaderCell,
  TableRow,
  TableWrap,
} from '@shared/ui';
import type { Application } from '@entities/application';
import type { Agent } from '@entities/agent';

export default function ApplicationDetailPage() {
  const { id } = useParams<{ id: string }>();
  const application = useApi<Application>(`/api/v1/applications/${id}`);
  const agents = useCursorPagination<Agent>(`/api/v1/applications/${id}/agents`, {
    initialPageSize: 200,
    enabled: Boolean(id),
  });
  const error = application.error ?? agents.error;
  if (application.isLoading)
    return (
      <div className="mt-16 flex justify-center">
        <Spinner className="size-6" />
      </div>
    );
  if (error || !application.data)
    return (
      <div>
        <Button variant="ghost" size="sm" className="mb-3" asChild>
          <Link href="/applications">
            <ArrowLeft aria-hidden />
            Back to applications
          </Link>
        </Button>
        <Alert severity="error">
          {error instanceof Error ? error.message : 'Application not found'}
        </Alert>
      </div>
    );
  const value = application.data;
  return (
    <div>
      <Button variant="ghost" size="sm" className="mb-2 -ml-2" asChild>
        <Link href="/applications">
          <ArrowLeft aria-hidden />
          Back to applications
        </Link>
      </Button>
      <PageHeader
        title={value.metadata.name}
        subtitle={`Service namespace: ${value.spec.namespace || 'default'}`}
        actions={
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label="Refresh"
            onClick={() => {
              void application.mutate();
              agents.refresh();
            }}
          >
            <RefreshCw aria-hidden />
          </Button>
        }
      />
      <Card className="mb-3">
        <CardContent className="flex flex-wrap gap-6 pt-4">
          <div>
            <p className="text-xs text-muted-foreground">Versions</p>
            <div className="mt-1 flex flex-wrap gap-1">
              {value.spec.versions?.map((version) => (
                <Badge key={version} variant="outline">
                  {version}
                </Badge>
              )) || 'none'}
            </div>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">Last seen</p>
            <TimeDisplay value={value.metadata.lastSeenAt} />
          </div>
          <div>
            <p className="text-xs text-muted-foreground">Agent type</p>
            <p>{value.spec.agentType || '-'}</p>
          </div>
        </CardContent>
      </Card>
      <h2 className="mb-2 text-lg font-semibold">
        Agents ({value.status.agentInstanceUids.length})
      </h2>
      <TableWrap>
        <Table>
          <TableHead>
            <TableRow className="hover:bg-transparent">
              <TableHeaderCell>Instance UID</TableHeaderCell>
              <TableHeaderCell>Namespace</TableHeaderCell>
              <TableHeaderCell>Status</TableHeaderCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {agents.isLoading ? (
              <TableRow>
                <TableCell colSpan={3}>
                  <Spinner className="mx-auto size-5" />
                </TableCell>
              </TableRow>
            ) : (
              agents.items.map((agent) => (
                <TableRow key={agent.metadata.instanceUid}>
                  <TableCell className="font-mono text-xs">
                    <Link
                      className="text-primary hover:underline"
                      href={`/agents/${agent.metadata.instanceUid}`}
                    >
                      {agent.metadata.instanceUid}
                    </Link>
                  </TableCell>
                  <TableCell>{agent.metadata.namespace}</TableCell>
                  <TableCell>{agent.status.connected ? 'Connected' : 'Disconnected'}</TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableWrap>
      <PaginationFooter pagination={agents} />
    </div>
  );
}
