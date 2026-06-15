'use client';

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
  Tab,
  Tabs,
} from '@mui/material';
import { Refresh as RefreshIcon } from '@mui/icons-material';
import { useCallback, useEffect, useState } from 'react';
import { PageHeader } from '@shared/ui';
import { TimeDisplay } from '@shared/preferences';
import { api, type ListResponse } from '@shared/api';
import type { Host } from '@entities/host';
import type { Container } from '@entities/container';

// platformColor maps a Platform label to an MUI chip color so the deployment
// environment is visually scannable across the host/container tabs.
function platformColor(
  platform: string,
): 'default' | 'primary' | 'secondary' | 'info' | 'success' | 'warning' {
  switch (platform) {
    case 'kubernetes':
      return 'primary';
    case 'docker':
      return 'info';
    case 'vm':
      return 'success';
    case 'baremetal':
      return 'warning';
    case 'ecs':
      return 'secondary';
    default:
      return 'default';
  }
}

function PlatformChip({ platform }: { platform: string }) {
  return <Chip size="small" label={platform || 'unknown'} color={platformColor(platform)} />;
}

function dash(value: string | undefined): string {
  return value && value.length > 0 ? value : '-';
}

export default function PlatformPage() {
  const [tab, setTab] = useState(0);
  const [hosts, setHosts] = useState<Host[]>([]);
  const [containers, setContainers] = useState<Container[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchAll = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [hostsRes, containersRes] = await Promise.all([
        api.get<ListResponse<Host>>('/api/v1/hosts', { query: { limit: 200 } }),
        api.get<ListResponse<Container>>('/api/v1/containers', { query: { limit: 200 } }),
      ]);
      setHosts(hostsRes.items ?? []);
      setContainers(containersRes.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch platform inventory');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void fetchAll();
  }, [fetchAll]);

  return (
    <Box>
      <PageHeader
        title="Platform"
        actions={
          <IconButton color="primary" onClick={fetchAll}>
            <RefreshIcon />
          </IconButton>
        }
      />

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      <Tabs value={tab} onChange={(_, v) => setTab(v)} sx={{ mb: 2 }}>
        <Tab label={`Hosts (${hosts.length})`} />
        <Tab label={`Containers (${containers.length})`} />
      </Tabs>

      {tab === 0 && <HostsTable hosts={hosts} loading={loading} />}
      {tab === 1 && <ContainersTable containers={containers} loading={loading} />}
    </Box>
  );
}

function HostsTable({ hosts, loading }: { hosts: Host[]; loading: boolean }) {
  return (
    <TableContainer component={Paper}>
      <Table>
        <TableHead>
          <TableRow>
            <TableCell>ID</TableCell>
            <TableCell>Name</TableCell>
            <TableCell>Platform</TableCell>
            <TableCell>Arch</TableCell>
            <TableCell>OS</TableCell>
            <TableCell>Cloud</TableCell>
            <TableCell align="right">Agents</TableCell>
            <TableCell>Last Seen</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {loading ? (
            <TableRow>
              <TableCell colSpan={8} align="center">
                <CircularProgress size={24} />
              </TableCell>
            </TableRow>
          ) : hosts.length === 0 ? (
            <TableRow>
              <TableCell colSpan={8} align="center">
                No hosts discovered
              </TableCell>
            </TableRow>
          ) : (
            hosts.map((host) => (
              <TableRow key={host.metadata.id} hover>
                <TableCell sx={{ fontFamily: 'monospace', fontSize: 12 }}>
                  {host.metadata.id}
                </TableCell>
                <TableCell>{dash(host.metadata.name)}</TableCell>
                <TableCell>
                  <PlatformChip platform={host.spec.platform} />
                </TableCell>
                <TableCell>{dash(host.spec.arch)}</TableCell>
                <TableCell>{dash(host.spec.osType)}</TableCell>
                <TableCell>{dash(host.spec.cloudProvider)}</TableCell>
                <TableCell align="right">{host.status.agentInstanceUids?.length ?? 0}</TableCell>
                <TableCell>
                  <TimeDisplay value={host.metadata.lastSeenAt} />
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </TableContainer>
  );
}

function ContainersTable({ containers, loading }: { containers: Container[]; loading: boolean }) {
  return (
    <TableContainer component={Paper}>
      <Table>
        <TableHead>
          <TableRow>
            <TableCell>ID</TableCell>
            <TableCell>Name</TableCell>
            <TableCell>Platform</TableCell>
            <TableCell>Image</TableCell>
            <TableCell>Runtime</TableCell>
            <TableCell>Host</TableCell>
            <TableCell align="right">Agents</TableCell>
            <TableCell>Last Seen</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {loading ? (
            <TableRow>
              <TableCell colSpan={8} align="center">
                <CircularProgress size={24} />
              </TableCell>
            </TableRow>
          ) : containers.length === 0 ? (
            <TableRow>
              <TableCell colSpan={8} align="center">
                No containers discovered
              </TableCell>
            </TableRow>
          ) : (
            containers.map((container) => (
              <TableRow key={container.metadata.id} hover>
                <TableCell sx={{ fontFamily: 'monospace', fontSize: 12 }}>
                  {container.metadata.id}
                </TableCell>
                <TableCell>{dash(container.metadata.name)}</TableCell>
                <TableCell>
                  <PlatformChip platform={container.spec.platform} />
                </TableCell>
                <TableCell>{dash(container.spec.imageName)}</TableCell>
                <TableCell>{dash(container.spec.runtime)}</TableCell>
                <TableCell sx={{ fontFamily: 'monospace', fontSize: 12 }}>
                  {dash(container.spec.hostId)}
                </TableCell>
                <TableCell align="right">
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
    </TableContainer>
  );
}
