'use client';

import {
  Box,
  Card,
  CardActionArea,
  CardContent,
  Grid,
  Stack,
  Typography,
} from '@mui/material';
import {
  Computer as ComputerIcon,
  Group as GroupIcon,
  Cable as CableIcon,
  Dns as DnsIcon,
  Folder as FolderIcon,
  Inventory2 as PackageIcon,
  Tune as TuneIcon,
  VerifiedUser as CertIcon,
} from '@mui/icons-material';
import Link from 'next/link';
import { ReactNode, useCallback, useEffect, useState } from 'react';
import PageHeader from '@/components/PageHeader';
import { useNamespace } from '@/components/NamespaceProvider';
import { api } from '@/lib/api-client';
import type { ListResponse } from '@/lib/types';

interface TileSummary {
  total: number | null;
  detail?: string;
}

interface TileProps {
  title: string;
  icon: ReactNode;
  href: string;
  summary: TileSummary;
}

function Tile({ title, icon, href, summary }: TileProps) {
  return (
    <Card>
      <CardActionArea component={Link} href={href}>
        <CardContent>
          <Stack direction="row" alignItems="center" gap={2}>
            <Box color="primary.main" display="flex">
              {icon}
            </Box>
            <Box flexGrow={1}>
              <Typography variant="subtitle1">{title}</Typography>
              <Typography variant="h5">
                {summary.total === null ? '—' : summary.total}
              </Typography>
              {summary.detail && (
                <Typography variant="body2" color="text.secondary">
                  {summary.detail}
                </Typography>
              )}
            </Box>
          </Stack>
        </CardContent>
      </CardActionArea>
    </Card>
  );
}

async function count(path: string, query?: Record<string, unknown>): Promise<TileSummary> {
  try {
    const res = await api.get<ListResponse<unknown>>(path, {
      query: { limit: 200, ...(query as Record<string, string | number>) },
    });
    return {
      total: (res.items?.length ?? 0) + (res.metadata?.remainingItemCount ?? 0),
    };
  } catch {
    return { total: null };
  }
}

export default function DashboardPage() {
  const { namespace } = useNamespace();
  const [stats, setStats] = useState<Record<string, TileSummary>>({});

  const refresh = useCallback(async () => {
    const entries: Array<[string, Promise<TileSummary>]> = [
      ['agents', count(`/api/v1/namespaces/${namespace}/agents`)],
      ['agentgroups', count(`/api/v1/namespaces/${namespace}/agentgroups`)],
      ['connections', count(`/api/v1/namespaces/${namespace}/connections`)],
      ['agentpackages', count(`/api/v1/namespaces/${namespace}/agentpackages`)],
      ['agentremoteconfigs', count(`/api/v1/namespaces/${namespace}/agentremoteconfigs`)],
      ['certificates', count(`/api/v1/namespaces/${namespace}/certificates`)],
      ['namespaces', count('/api/v1/namespaces')],
      ['servers', count('/api/v1/servers')],
    ];
    const resolved = await Promise.all(entries.map(([, p]) => p));
    const map: Record<string, TileSummary> = {};
    entries.forEach(([key], i) => {
      map[key] = resolved[i];
    });
    setStats(map);
  }, [namespace]);

  // eslint-disable-next-line react-hooks/set-state-in-effect
  useEffect(() => { void refresh(); }, [refresh]);

  return (
    <Box>
      <PageHeader
        title="Dashboard"
        subtitle={`Overview for namespace "${namespace}"`}
      />

      <Grid container spacing={2}>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <Tile title="Agents" icon={<ComputerIcon fontSize="large" />} href="/agents" summary={stats.agents ?? { total: null }} />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <Tile title="Agent Groups" icon={<GroupIcon fontSize="large" />} href="/agentgroups" summary={stats.agentgroups ?? { total: null }} />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <Tile title="Connections" icon={<CableIcon fontSize="large" />} href="/connections" summary={stats.connections ?? { total: null }} />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <Tile title="Servers" icon={<DnsIcon fontSize="large" />} href="/servers" summary={stats.servers ?? { total: null }} />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <Tile title="Namespaces" icon={<FolderIcon fontSize="large" />} href="/namespaces" summary={stats.namespaces ?? { total: null }} />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <Tile title="Agent Packages" icon={<PackageIcon fontSize="large" />} href="/agentpackages" summary={stats.agentpackages ?? { total: null }} />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <Tile title="Remote Configs" icon={<TuneIcon fontSize="large" />} href="/agentremoteconfigs" summary={stats.agentremoteconfigs ?? { total: null }} />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <Tile title="Certificates" icon={<CertIcon fontSize="large" />} href="/certificates" summary={stats.certificates ?? { total: null }} />
        </Grid>
      </Grid>
    </Box>
  );
}
