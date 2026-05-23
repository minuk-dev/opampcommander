'use client';

import {
  Box,
  Card,
  CardActionArea,
  CardContent,
  CardHeader,
  Chip,
  CircularProgress,
  Divider,
  Grid,
  IconButton,
  LinearProgress,
  Stack,
  Tooltip,
  Typography,
} from '@mui/material';
import {
  Computer as ComputerIcon,
  Group as GroupIcon,
  Cable as CableIcon,
  Dns as DnsIcon,
  Inventory2 as PackageIcon,
  Tune as TuneIcon,
  VerifiedUser as CertIcon,
  ArrowForward as ArrowForwardIcon,
  Refresh as RefreshIcon,
  CheckCircle as CheckCircleIcon,
  HighlightOff as OffIcon,
  HealthAndSafety as HealthIcon,
} from '@mui/icons-material';
import Link from 'next/link';
import { type ReactNode, useCallback, useEffect, useState } from 'react';
import PageHeader from '@/components/PageHeader';
import { useNamespace } from '@/components/NamespaceProvider';
import { api } from '@/lib/api-client';
import type { Agent, AgentGroup, Connection, ListResponse, Server, VersionInfo } from '@/lib/types';

interface QuadrantProps {
  title: string;
  href: string;
  icon: ReactNode;
  loading?: boolean;
  children: ReactNode;
}

function Quadrant({ title, href, icon, loading, children }: QuadrantProps) {
  return (
    <Card variant="outlined" sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <CardHeader
        avatar={<Box sx={{ color: 'primary.main', display: 'flex' }}>{icon}</Box>}
        title={
          <Stack direction="row" alignItems="center" gap={1}>
            <Typography variant="h6">{title}</Typography>
            {loading && <CircularProgress size={14} />}
          </Stack>
        }
        action={
          <Tooltip title={`Open ${title}`}>
            <IconButton component={Link} href={href} aria-label={`view ${title}`}>
              <ArrowForwardIcon />
            </IconButton>
          </Tooltip>
        }
        sx={{ pb: 0 }}
      />
      <Divider sx={{ mx: 2, mt: 1 }} />
      <CardContent sx={{ flexGrow: 1 }}>{children}</CardContent>
    </Card>
  );
}

function Stat({
  label,
  value,
  detail,
  color,
}: {
  label: string;
  value: ReactNode;
  detail?: ReactNode;
  color?: 'success' | 'warning' | 'error' | 'default';
}) {
  return (
    <Box>
      <Typography variant="caption" color="text.secondary">
        {label}
      </Typography>
      <Typography
        variant="h5"
        sx={{
          color:
            color === 'success'
              ? 'success.main'
              : color === 'warning'
                ? 'warning.main'
                : color === 'error'
                  ? 'error.main'
                  : 'text.primary',
        }}
      >
        {value}
      </Typography>
      {detail && (
        <Typography variant="caption" color="text.secondary">
          {detail}
        </Typography>
      )}
    </Box>
  );
}

interface DashboardData {
  agents: Agent[];
  agentTotal: number;
  groups: AgentGroup[];
  connections: Connection[];
  servers: Server[];
  packages: number | null;
  remoteConfigs: number | null;
  certificates: number | null;
  version: VersionInfo | null;
}

async function safeList<T>(
  path: string,
  limit = 200,
): Promise<{ items: T[]; total: number | null }> {
  try {
    const res = await api.get<ListResponse<T>>(path, { query: { limit } });
    const items = res.items ?? [];
    return {
      items,
      total: items.length + (res.metadata?.remainingItemCount ?? 0),
    };
  } catch {
    return { items: [], total: null };
  }
}

export default function DashboardPage() {
  const { namespace } = useNamespace();
  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(async () => {
    setLoading(true);
    const [agents, groups, conns, servers, packages, configs, certs, version] = await Promise.all([
      safeList<Agent>(`/api/v1/namespaces/${namespace}/agents`),
      safeList<AgentGroup>(`/api/v1/namespaces/${namespace}/agentgroups`),
      safeList<Connection>(`/api/v1/namespaces/${namespace}/connections`),
      safeList<Server>('/api/v1/servers'),
      safeList<unknown>(`/api/v1/namespaces/${namespace}/agentpackages`),
      safeList<unknown>(`/api/v1/namespaces/${namespace}/agentremoteconfigs`),
      safeList<unknown>(`/api/v1/namespaces/${namespace}/certificates`),
      api.get<VersionInfo>('/api/v1/version').catch(() => null),
    ]);
    setData({
      agents: agents.items,
      agentTotal: agents.total ?? agents.items.length,
      groups: groups.items,
      connections: conns.items,
      servers: servers.items,
      packages: packages.total,
      remoteConfigs: configs.total,
      certificates: certs.total,
      version,
    });
    setLoading(false);
  }, [namespace]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void refresh();
  }, [refresh]);

  const connectedCount = data?.agents.filter((a) => a.status.connected).length ?? 0;
  const healthyCount = data?.agents.filter((a) => a.status.componentHealth?.healthy).length ?? 0;
  const aliveConns = data?.connections.filter((c) => c.alive).length ?? 0;
  const aliveServers =
    data?.servers.filter((s) =>
      s.conditions?.some((c) => c.type === 'Alive' && c.status === 'True'),
    ).length ?? 0;

  const topGroups = (data?.groups ?? [])
    .slice()
    .sort((a, b) => b.status.numAgents - a.status.numAgents)
    .slice(0, 5);

  return (
    <Box>
      <PageHeader
        title="Dashboard"
        subtitle={`Overview for namespace "${namespace}"`}
        actions={
          <Tooltip title="Refresh">
            <IconButton color="primary" onClick={refresh} aria-label="refresh">
              <RefreshIcon />
            </IconButton>
          </Tooltip>
        }
      />

      <Grid container spacing={2} sx={{ alignItems: 'stretch' }}>
        {/* Quadrant 1 — Agents */}
        <Grid size={{ xs: 12, md: 6 }}>
          <Quadrant
            title="Agents"
            href="/agents"
            icon={<ComputerIcon fontSize="large" />}
            loading={loading}
          >
            <Grid container spacing={2}>
              <Grid size={{ xs: 4 }}>
                <Stat label="Total" value={data?.agentTotal ?? '—'} />
              </Grid>
              <Grid size={{ xs: 4 }}>
                <Stat
                  label="Connected"
                  value={`${connectedCount}/${data?.agents.length ?? 0}`}
                  color={connectedCount > 0 ? 'success' : 'default'}
                />
              </Grid>
              <Grid size={{ xs: 4 }}>
                <Stat
                  label="Healthy"
                  value={`${healthyCount}/${data?.agents.length ?? 0}`}
                  color={
                    data && data.agents.length > 0 && healthyCount === data.agents.length
                      ? 'success'
                      : 'warning'
                  }
                />
              </Grid>
            </Grid>
            <Box sx={{ mt: 2 }}>
              <Typography variant="caption" color="text.secondary">
                Health on the current page
              </Typography>
              <LinearProgress
                variant="determinate"
                value={
                  data && data.agents.length > 0 ? (healthyCount / data.agents.length) * 100 : 0
                }
                color={
                  data && data.agents.length > 0 && healthyCount === data.agents.length
                    ? 'success'
                    : 'warning'
                }
                sx={{ height: 6, mt: 0.5, borderRadius: 1 }}
              />
            </Box>
            {data && data.agents.length === 0 && (
              <Typography variant="body2" color="text.secondary" mt={2}>
                No agents reported yet in this namespace.
              </Typography>
            )}
          </Quadrant>
        </Grid>

        {/* Quadrant 2 — Agent Groups */}
        <Grid size={{ xs: 12, md: 6 }}>
          <Quadrant
            title="Agent Groups"
            href="/agentgroups"
            icon={<GroupIcon fontSize="large" />}
            loading={loading}
          >
            {topGroups.length === 0 ? (
              <Typography variant="body2" color="text.secondary">
                {loading ? 'Loading…' : 'No agent groups yet.'}
              </Typography>
            ) : (
              <Stack spacing={1}>
                {topGroups.map((g) => (
                  <Card
                    key={g.metadata.name}
                    variant="outlined"
                    sx={{
                      transition: 'border-color 0.1s',
                      '&:hover': { borderColor: 'primary.main' },
                    }}
                  >
                    <CardActionArea
                      component={Link}
                      href={`/agents?agentGroup=${encodeURIComponent(g.metadata.name)}`}
                    >
                      <Box
                        sx={{
                          p: 1.5,
                          display: 'flex',
                          alignItems: 'center',
                          gap: 1,
                        }}
                      >
                        <Box sx={{ flexGrow: 1, minWidth: 0 }}>
                          <Typography
                            variant="subtitle2"
                            sx={{
                              overflow: 'hidden',
                              textOverflow: 'ellipsis',
                              whiteSpace: 'nowrap',
                            }}
                          >
                            {g.metadata.name}
                          </Typography>
                          <Typography variant="caption" color="text.secondary">
                            priority {g.spec.priority} · {g.status.numHealthyAgents}/
                            {g.status.numAgents} healthy
                          </Typography>
                        </Box>
                        <Chip
                          size="small"
                          label={`${g.status.numConnectedAgents}/${g.status.numAgents}`}
                          color={
                            g.status.numAgents > 0 &&
                            g.status.numConnectedAgents === g.status.numAgents
                              ? 'success'
                              : 'default'
                          }
                          variant="outlined"
                        />
                      </Box>
                    </CardActionArea>
                  </Card>
                ))}
              </Stack>
            )}
          </Quadrant>
        </Grid>

        {/* Quadrant 3 — Resources */}
        <Grid size={{ xs: 12, md: 6 }}>
          <Quadrant
            title="Resources"
            href="/agentpackages"
            icon={<PackageIcon fontSize="large" />}
            loading={loading}
          >
            <Stack
              direction={{ xs: 'column', sm: 'row' }}
              divider={<Divider orientation="vertical" flexItem />}
              spacing={2}
              justifyContent="space-around"
            >
              {[
                {
                  href: '/agentpackages',
                  icon: <PackageIcon color="primary" />,
                  total: data?.packages,
                  label: 'Packages',
                },
                {
                  href: '/agentremoteconfigs',
                  icon: <TuneIcon color="primary" />,
                  total: data?.remoteConfigs,
                  label: 'Remote Configs',
                },
                {
                  href: '/certificates',
                  icon: <CertIcon color="primary" />,
                  total: data?.certificates,
                  label: 'Certificates',
                },
              ].map((r) => (
                <Box
                  key={r.label}
                  component={Link}
                  href={r.href}
                  sx={{
                    textDecoration: 'none',
                    color: 'inherit',
                    display: 'block',
                    textAlign: 'center',
                    flex: 1,
                    borderRadius: 1,
                    px: 1,
                    py: 0.5,
                    '&:hover': { bgcolor: 'action.hover' },
                  }}
                >
                  {r.icon}
                  <Typography variant="h5">{r.total ?? '—'}</Typography>
                  <Typography variant="caption" color="text.secondary">
                    {r.label}
                  </Typography>
                </Box>
              ))}
            </Stack>
          </Quadrant>
        </Grid>

        {/* Quadrant 4 — Cluster */}
        <Grid size={{ xs: 12, md: 6 }}>
          <Quadrant
            title="Cluster"
            href="/servers"
            icon={<DnsIcon fontSize="large" />}
            loading={loading}
          >
            <Grid container spacing={2}>
              <Grid size={{ xs: 6 }}>
                <Stack
                  component={Link}
                  href="/connections"
                  direction="row"
                  alignItems="center"
                  gap={1}
                  sx={{
                    textDecoration: 'none',
                    color: 'inherit',
                    borderRadius: 1,
                    p: 1,
                    '&:hover': { bgcolor: 'action.hover' },
                  }}
                >
                  <CableIcon color="primary" />
                  <Box>
                    <Typography variant="h6">
                      {aliveConns}/{data?.connections.length ?? 0}
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                      Connections (alive)
                    </Typography>
                  </Box>
                </Stack>
              </Grid>
              <Grid size={{ xs: 6 }}>
                <Stack
                  component={Link}
                  href="/servers"
                  direction="row"
                  alignItems="center"
                  gap={1}
                  sx={{
                    textDecoration: 'none',
                    color: 'inherit',
                    borderRadius: 1,
                    p: 1,
                    '&:hover': { bgcolor: 'action.hover' },
                  }}
                >
                  {data && aliveServers > 0 ? (
                    <CheckCircleIcon color="success" />
                  ) : (
                    <OffIcon color="warning" />
                  )}
                  <Box>
                    <Typography variant="h6">
                      {aliveServers}/{data?.servers.length ?? 0}
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                      Servers (alive)
                    </Typography>
                  </Box>
                </Stack>
              </Grid>
              <Grid size={{ xs: 12 }}>
                <Divider sx={{ my: 1 }} />
                <Stack direction="row" alignItems="center" gap={1}>
                  <HealthIcon fontSize="small" color="action" />
                  <Typography variant="caption" color="text.secondary">
                    Server build:
                  </Typography>
                  <Typography variant="caption" sx={{ fontFamily: 'monospace' }}>
                    {data?.version?.gitVersion ?? '—'}
                  </Typography>
                </Stack>
              </Grid>
            </Grid>
          </Quadrant>
        </Grid>
      </Grid>
    </Box>
  );
}
