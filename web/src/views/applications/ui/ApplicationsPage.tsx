'use client';

import { RefreshCw } from 'lucide-react';
import Link from 'next/link';
import { useMemo, useState } from 'react';
import { useApi, type ListResponse } from '@shared/api';
import { EMPTY_LIST_FILTERS, hasListFilters, listFilterQuery, type ListFilters } from '@shared/lib';
import { TimeDisplay } from '@shared/preferences';
import {
  Alert,
  Badge,
  Button,
  ListFilterBar,
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
import type { Application } from '@entities/application';

const PAGE_LIMIT = 200;

export default function ApplicationsPage() {
  const [filters, setFilters] = useState<ListFilters>({
    ...EMPTY_LIST_FILTERS,
    nameMatch: 'prefix',
  });
  const query = useMemo(() => listFilterQuery(filters), [filters]);
  const {
    data,
    error: fetchError,
    isLoading,
    isValidating,
    mutate,
  } = useApi<ListResponse<Application>>(['/api/v1/applications', { limit: PAGE_LIMIT, ...query }]);
  const applications = data?.items ?? [];
  const error =
    fetchError instanceof Error
      ? fetchError.message
      : fetchError
        ? 'Failed to fetch applications'
        : null;

  return (
    <div>
      <PageHeader
        title="Applications"
        subtitle="Services discovered from agent attributes"
        actions={
          <Button variant="ghost" size="icon-sm" aria-label="Refresh" onClick={() => void mutate()}>
            <RefreshCw className={isValidating ? 'animate-spin' : ''} aria-hidden />
          </Button>
        }
      />
      <ListFilterBar
        value={filters}
        onChange={setFilters}
        namePlaceholder="Application name starts with…"
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
              <TableHeaderCell>Name</TableHeaderCell>
              <TableHeaderCell>Namespace</TableHeaderCell>
              <TableHeaderCell>Versions</TableHeaderCell>
              <TableHeaderCell>Type</TableHeaderCell>
              <TableHeaderCell className="text-right">Agents</TableHeaderCell>
              <TableHeaderCell>Last Seen</TableHeaderCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {isLoading || applications.length === 0 ? (
              <TableRow className="hover:bg-transparent">
                <TableCell colSpan={6} className="py-8 text-center text-muted-foreground">
                  {isLoading ? (
                    <Spinner className="mx-auto size-5" />
                  ) : hasListFilters(filters) ? (
                    'No applications match the filters'
                  ) : (
                    'No applications discovered'
                  )}
                </TableCell>
              </TableRow>
            ) : (
              applications.map((application) => (
                <TableRow key={application.metadata.id}>
                  <TableCell>
                    <Link
                      className="font-medium text-primary hover:underline"
                      href={`/applications/${application.metadata.id}`}
                    >
                      {application.metadata.name}
                    </Link>
                  </TableCell>
                  <TableCell>{application.spec.namespace || '-'}</TableCell>
                  <TableCell className="space-x-1">
                    {application.spec.versions?.map((version) => (
                      <Badge key={version} variant="outline">
                        {version}
                      </Badge>
                    )) || '-'}
                  </TableCell>
                  <TableCell>{application.spec.agentType || '-'}</TableCell>
                  <TableCell className="tnum text-right">
                    {application.status.agentInstanceUids.length}
                  </TableCell>
                  <TableCell>
                    <TimeDisplay value={application.metadata.lastSeenAt} />
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
