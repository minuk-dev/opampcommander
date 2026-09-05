'use client';

import { RefreshCw } from 'lucide-react';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { api, type ListResponse } from '@shared/api';
import {
  cn,
  EMPTY_LIST_FILTERS,
  hasListFilters,
  listFilterQuery,
  type ListFilters,
} from '@shared/lib';
import { TimeDisplay } from '@shared/preferences';
import {
  Alert,
  Badge,
  Button,
  ListFilterBar,
  PageHeader,
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
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@shared/ui';
import type { Host } from '@entities/host';
import type { Container } from '@entities/container';

// The page size both tabs request. Hosts and containers are filtered by the
// server, so a narrowed view is a smaller query rather than a larger fetch.
const PAGE_LIMIT = 200;

// Each Platform label gets its own badge tone so the deployment environment is
// visually scannable across the host/container tabs.
const PLATFORM_VARIANTS = {
  kubernetes: 'primary',
  docker: 'primary',
  vm: 'success',
  baremetal: 'warning',
  ecs: 'outline',
} as const;

function PlatformBadge({ platform }: { platform: string }) {
  const variant = PLATFORM_VARIANTS[platform as keyof typeof PLATFORM_VARIANTS] ?? 'muted';
  return <Badge variant={variant}>{platform || 'unknown'}</Badge>;
}

function dash(value: string | undefined): string {
  return value && value.length > 0 ? value : '-';
}

// A platform is one of a small closed set, so it gets a dropdown that writes the
// `spec.platform` field selector rather than a free-text field the user has to
// spell exactly. "any" removes the requirement.
function PlatformSelect({
  value,
  onChange,
}: {
  value: ListFilters;
  onChange: (next: ListFilters) => void;
}) {
  const selected = value.fieldSelector.startsWith('spec.platform=')
    ? value.fieldSelector.slice('spec.platform='.length)
    : 'any';
  return (
    <div className="mt-2 flex items-center gap-2">
      <span className="text-xs text-muted-foreground">Platform</span>
      <Select
        value={selected}
        onValueChange={(next) =>
          onChange({ ...value, fieldSelector: next === 'any' ? '' : `spec.platform=${next}` })
        }
      >
        <SelectTrigger className="w-40">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="any">Any</SelectItem>
          {Object.keys(PLATFORM_VARIANTS).map((platform) => (
            <SelectItem key={platform} value={platform}>
              {platform}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}

function TableState({
  colSpan,
  loading,
  empty,
}: {
  colSpan: number;
  loading: boolean;
  empty: string;
}) {
  return (
    <TableRow className="hover:bg-transparent">
      <TableCell colSpan={colSpan} className="py-8 text-center text-muted-foreground">
        {loading ? <Spinner className="mx-auto size-5" /> : empty}
      </TableCell>
    </TableRow>
  );
}

export default function PlatformPage() {
  const [hosts, setHosts] = useState<Host[]>([]);
  const [containers, setContainers] = useState<Container[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  // Each tab filters its own collection; the two are fetched together, so a
  // change to either set of filters refetches both.
  const [hostFilters, setHostFilters] = useState<ListFilters>(EMPTY_LIST_FILTERS);
  const [containerFilters, setContainerFilters] = useState<ListFilters>(EMPTY_LIST_FILTERS);

  // Memoised on the filter state, whose identity only changes when a filter is
  // applied, so fetchAll is stable between renders and the effect below runs
  // once per filter change rather than once per render.
  const hostQuery = useMemo(() => listFilterQuery(hostFilters), [hostFilters]);
  const containerQuery = useMemo(() => listFilterQuery(containerFilters), [containerFilters]);

  const fetchAll = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [hostsRes, containersRes] = await Promise.all([
        api.get<ListResponse<Host>>('/api/v1/hosts', {
          query: { limit: PAGE_LIMIT, ...hostQuery },
        }),
        api.get<ListResponse<Container>>('/api/v1/containers', {
          query: { limit: PAGE_LIMIT, ...containerQuery },
        }),
      ]);
      setHosts(hostsRes.items ?? []);
      setContainers(containersRes.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch platform inventory');
    } finally {
      setLoading(false);
    }
  }, [hostQuery, containerQuery]);

  useEffect(() => {
    void fetchAll();
  }, [fetchAll]);

  return (
    <div>
      <PageHeader
        title="Platform"
        subtitle="Hosts and containers discovered from agent attributes"
        actions={
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label="Refresh"
            onClick={() => void fetchAll()}
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

      <Tabs defaultValue="hosts">
        <TabsList className="mb-3">
          <TabsTrigger value="hosts">Hosts ({hosts.length})</TabsTrigger>
          <TabsTrigger value="containers">Containers ({containers.length})</TabsTrigger>
        </TabsList>
        <TabsContent value="hosts">
          <ListFilterBar
            value={hostFilters}
            onChange={setHostFilters}
            namePlaceholder="Host name contains…"
          >
            <PlatformSelect value={hostFilters} onChange={setHostFilters} />
          </ListFilterBar>
          <HostsTable hosts={hosts} loading={loading} filtered={hasListFilters(hostFilters)} />
        </TabsContent>
        <TabsContent value="containers">
          <ListFilterBar
            value={containerFilters}
            onChange={setContainerFilters}
            namePlaceholder="Container name contains…"
          >
            <PlatformSelect value={containerFilters} onChange={setContainerFilters} />
          </ListFilterBar>
          <ContainersTable
            containers={containers}
            loading={loading}
            filtered={hasListFilters(containerFilters)}
          />
        </TabsContent>
      </Tabs>
    </div>
  );
}

function HostsTable({
  hosts,
  loading,
  filtered,
}: {
  hosts: Host[];
  loading: boolean;
  filtered: boolean;
}) {
  return (
    <TableWrap>
      <Table>
        <TableHead>
          <TableRow className="hover:bg-transparent">
            <TableHeaderCell>ID</TableHeaderCell>
            <TableHeaderCell>Name</TableHeaderCell>
            <TableHeaderCell>Platform</TableHeaderCell>
            <TableHeaderCell>Arch</TableHeaderCell>
            <TableHeaderCell>OS</TableHeaderCell>
            <TableHeaderCell>Cloud</TableHeaderCell>
            <TableHeaderCell className="text-right">Agents</TableHeaderCell>
            <TableHeaderCell>Last Seen</TableHeaderCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {loading || hosts.length === 0 ? (
            <TableState
              colSpan={8}
              loading={loading}
              empty={filtered ? 'No hosts match the filters' : 'No hosts discovered'}
            />
          ) : (
            hosts.map((host) => (
              <TableRow key={host.metadata.id}>
                <TableCell className="font-mono text-xs">{host.metadata.id}</TableCell>
                <TableCell>{dash(host.metadata.name)}</TableCell>
                <TableCell>
                  <PlatformBadge platform={host.spec.platform} />
                </TableCell>
                <TableCell>{dash(host.spec.arch)}</TableCell>
                <TableCell>{dash(host.spec.osType)}</TableCell>
                <TableCell>{dash(host.spec.cloudProvider)}</TableCell>
                <TableCell className="tnum text-right">
                  {host.status.agentInstanceUids?.length ?? 0}
                </TableCell>
                <TableCell>
                  <TimeDisplay value={host.metadata.lastSeenAt} />
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </TableWrap>
  );
}

function ContainersTable({
  containers,
  loading,
  filtered,
}: {
  containers: Container[];
  loading: boolean;
  filtered: boolean;
}) {
  return (
    <TableWrap>
      <Table>
        <TableHead>
          <TableRow className="hover:bg-transparent">
            <TableHeaderCell>ID</TableHeaderCell>
            <TableHeaderCell>Name</TableHeaderCell>
            <TableHeaderCell>Platform</TableHeaderCell>
            <TableHeaderCell>Image</TableHeaderCell>
            <TableHeaderCell>Runtime</TableHeaderCell>
            <TableHeaderCell>Host</TableHeaderCell>
            <TableHeaderCell className="text-right">Agents</TableHeaderCell>
            <TableHeaderCell>Last Seen</TableHeaderCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {loading || containers.length === 0 ? (
            <TableState
              colSpan={8}
              loading={loading}
              empty={filtered ? 'No containers match the filters' : 'No containers discovered'}
            />
          ) : (
            containers.map((container) => (
              <TableRow key={container.metadata.id}>
                <TableCell className="font-mono text-xs">{container.metadata.id}</TableCell>
                <TableCell>{dash(container.metadata.name)}</TableCell>
                <TableCell>
                  <PlatformBadge platform={container.spec.platform} />
                </TableCell>
                <TableCell>{dash(container.spec.imageName)}</TableCell>
                <TableCell>{dash(container.spec.runtime)}</TableCell>
                <TableCell className="font-mono text-xs">{dash(container.spec.hostId)}</TableCell>
                <TableCell className="tnum text-right">
                  {container.status.agentInstanceUids?.length ?? 0}
                </TableCell>
                <TableCell>
                  <TimeDisplay value={container.metadata.lastSeenAt} />
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </TableWrap>
  );
}
