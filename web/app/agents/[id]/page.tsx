'use client';

import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Divider,
  Grid,
  Stack,
  Tab,
  Tabs,
  ToggleButton,
  ToggleButtonGroup,
  Typography,
} from '@mui/material';
import {
  Edit as EditIcon,
  ArrowBack as ArrowBackIcon,
  Refresh as RefreshIcon,
  RestartAlt as RestartIcon,
} from '@mui/icons-material';
import { useParams, useRouter } from 'next/navigation';
import { useCallback, useEffect, useMemo, useState } from 'react';
import PageHeader from '@/components/PageHeader';
import JsonBlock from '@/components/JsonBlock';
import { useNamespace } from '@/components/NamespaceProvider';
import { api } from '@/lib/api-client';
import { toYAML } from '@/lib/yaml';
import type { Agent } from '@/lib/types';
import AgentEditDialog from './AgentEditDialog';

function capabilityNames(bitmask: number | undefined): string[] {
  if (!bitmask) return [];
  const table: Array<[number, string]> = [
    [1, 'ReportsStatus'],
    [2, 'AcceptsRemoteConfig'],
    [4, 'ReportsEffectiveConfig'],
    [8, 'AcceptsPackages'],
    [16, 'ReportsPackageStatuses'],
    [32, 'ReportsOwnTraces'],
    [64, 'ReportsOwnMetrics'],
    [128, 'ReportsOwnLogs'],
    [256, 'AcceptsOpAMPConnectionSettings'],
    [512, 'AcceptsOtherConnectionSettings'],
    [1024, 'AcceptsRestartCommand'],
    [2048, 'ReportsHealth'],
    [4096, 'ReportsRemoteConfig'],
    [8192, 'ReportsHeartbeat'],
    [16384, 'ReportsAvailableComponents'],
  ];
  return table.filter(([b]) => (bitmask & b) !== 0).map(([, name]) => name);
}

export default function AgentDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const { namespace } = useNamespace();
  const [agent, setAgent] = useState<Agent | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [tab, setTab] = useState(0);
  const [rawFormat, setRawFormat] = useState<'json' | 'yaml'>('json');
  const [editOpen, setEditOpen] = useState(false);
  const [restartBusy, setRestartBusy] = useState(false);

  const fetchAgent = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await api.get<Agent>(
        `/api/v1/namespaces/${namespace}/agents/${params.id}`,
      );
      setAgent(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch agent');
    } finally {
      setLoading(false);
    }
  }, [namespace, params.id]);

  useEffect(() => {
    void fetchAgent();
  }, [fetchAgent]);

  const requestRestart = async () => {
    if (!agent) return;
    setRestartBusy(true);
    try {
      const next: Agent = {
        ...agent,
        spec: {
          ...(agent.spec ?? {}),
          restartRequiredAt: new Date().toISOString(),
        },
      };
      const updated = await api.put<Agent>(
        `/api/v1/namespaces/${namespace}/agents/${params.id}`,
        next,
      );
      setAgent(updated);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to set restart');
    } finally {
      setRestartBusy(false);
    }
  };

  const capabilities = useMemo(
    () => capabilityNames(agent?.metadata.capabilities),
    [agent?.metadata.capabilities],
  );

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" mt={6}>
        <CircularProgress />
      </Box>
    );
  }

  if (error || !agent) {
    return (
      <Box>
        <Button startIcon={<ArrowBackIcon />} onClick={() => router.back()} sx={{ mb: 2 }}>
          Back
        </Button>
        <Alert severity="error">{error || 'Agent not found'}</Alert>
      </Box>
    );
  }

  return (
    <Box>
      <Button startIcon={<ArrowBackIcon />} onClick={() => router.push('/agents')} sx={{ mb: 2 }}>
        Back to agents
      </Button>
      <PageHeader
        title={agent.metadata.instanceUid}
        subtitle={`Namespace: ${agent.metadata.namespace}`}
        actions={
          <>
            <Button startIcon={<RefreshIcon />} onClick={fetchAgent}>
              Refresh
            </Button>
            <Button
              startIcon={<RestartIcon />}
              onClick={requestRestart}
              disabled={restartBusy}
            >
              Request restart
            </Button>
            <Button
              startIcon={<EditIcon />}
              variant="contained"
              onClick={() => setEditOpen(true)}
            >
              Edit spec
            </Button>
          </>
        }
      />

      <Grid container spacing={2} sx={{ mb: 2 }}>
        <Grid size={{ xs: 12, md: 4 }}>
          <Card>
            <CardContent>
              <Typography variant="overline" color="text.secondary">
                Connection
              </Typography>
              <Stack direction="row" gap={1} mt={1}>
                <Chip
                  label={agent.status.connected ? 'Connected' : 'Disconnected'}
                  color={agent.status.connected ? 'success' : 'default'}
                  size="small"
                />
                {agent.status.connectionType && (
                  <Chip label={agent.status.connectionType} size="small" />
                )}
              </Stack>
              <Typography variant="body2" mt={1}>
                Last reported: {agent.status.lastReportedAt || '—'}
              </Typography>
              <Typography variant="body2">
                Sequence #: {agent.status.sequenceNum ?? '—'}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid size={{ xs: 12, md: 4 }}>
          <Card>
            <CardContent>
              <Typography variant="overline" color="text.secondary">
                Health
              </Typography>
              <Stack direction="row" gap={1} mt={1}>
                <Chip
                  label={agent.status.componentHealth?.healthy ? 'Healthy' : 'Unhealthy'}
                  color={agent.status.componentHealth?.healthy ? 'success' : 'warning'}
                  size="small"
                />
                {agent.status.componentHealth?.status && (
                  <Chip label={agent.status.componentHealth.status} size="small" />
                )}
              </Stack>
              {agent.status.componentHealth?.lastError && (
                <Typography variant="body2" color="error" mt={1}>
                  {agent.status.componentHealth.lastError}
                </Typography>
              )}
              <Typography variant="body2" mt={1}>
                Started: {agent.status.componentHealth?.startTime || '—'}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid size={{ xs: 12, md: 4 }}>
          <Card>
            <CardContent>
              <Typography variant="overline" color="text.secondary">
                Capabilities
              </Typography>
              <Box mt={1} display="flex" flexWrap="wrap" gap={0.5}>
                {capabilities.length === 0 ? (
                  <Typography variant="body2" color="text.secondary">
                    None
                  </Typography>
                ) : (
                  capabilities.map((c) => (
                    <Chip key={c} label={c} size="small" variant="outlined" />
                  ))
                )}
              </Box>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      <Card>
        <Tabs value={tab} onChange={(_, v) => setTab(v)}>
          <Tab label="Description" />
          <Tab label="Effective config" />
          <Tab label="Spec" />
          <Tab label="Conditions" />
          <Tab label="Raw" />
        </Tabs>
        <Divider />
        <CardContent>
          {tab === 0 && (
            <Stack spacing={2}>
              <JsonBlock
                title="Identifying attributes"
                value={agent.metadata.description?.identifyingAttributes ?? {}}
              />
              <JsonBlock
                title="Non-identifying attributes"
                value={agent.metadata.description?.nonIdentifyingAttributes ?? {}}
              />
              <JsonBlock
                title="Custom capabilities"
                value={agent.metadata.customCapabilities?.capabilities ?? []}
              />
            </Stack>
          )}
          {tab === 1 && (
            <Stack spacing={2}>
              {Object.entries(
                agent.status.effectiveConfig?.configMap.configMap ?? {},
              ).length === 0 ? (
                <Typography color="text.secondary">No effective config reported.</Typography>
              ) : (
                Object.entries(
                  agent.status.effectiveConfig?.configMap.configMap ?? {},
                ).map(([name, file]) => (
                  <JsonBlock key={name} title={`${name} (${file.contentType})`} value={file.body} />
                ))
              )}
            </Stack>
          )}
          {tab === 2 && <JsonBlock value={agent.spec ?? {}} />}
          {tab === 3 && <JsonBlock value={agent.status.conditions ?? []} />}
          {tab === 4 && (
            <Stack spacing={2}>
              <Stack direction="row" justifyContent="flex-end">
                <ToggleButtonGroup
                  size="small"
                  exclusive
                  value={rawFormat}
                  onChange={(_, v: 'json' | 'yaml' | null) => v && setRawFormat(v)}
                  aria-label="raw format"
                >
                  <ToggleButton value="json">JSON</ToggleButton>
                  <ToggleButton value="yaml">YAML</ToggleButton>
                </ToggleButtonGroup>
              </Stack>
              {rawFormat === 'json' ? (
                <JsonBlock value={agent} />
              ) : (
                <JsonBlock value={toYAML(agent)} />
              )}
            </Stack>
          )}
        </CardContent>
      </Card>

      <AgentEditDialog
        open={editOpen}
        agent={agent}
        onClose={() => setEditOpen(false)}
        onSaved={(saved) => {
          setAgent(saved);
          setEditOpen(false);
        }}
      />
    </Box>
  );
}
