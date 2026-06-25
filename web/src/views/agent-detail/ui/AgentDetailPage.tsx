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
  Typography,
} from '@mui/material';
import {
  Edit as EditIcon,
  ArrowBack as ArrowBackIcon,
  Refresh as RefreshIcon,
  RestartAlt as RestartIcon,
  Delete as DeleteIcon,
} from '@mui/icons-material';
import { useParams, useRouter, useSearchParams } from 'next/navigation';
import { Suspense, useCallback, useEffect, useMemo, useState } from 'react';
import { PageHeader, ConfirmDialog, JsonBlock } from '@shared/ui';
import { TimeDisplay } from '@shared/preferences';
import { ReconcileButton } from '@features/reconcile';
import { useNamespace } from '@entities/namespace';
import { api, useApi } from '@shared/api';
import {
  agentDeleteConfirmMessage,
  agentTypeLabel,
  capabilityNames,
  deleteAgent,
  isOtelCollector,
  type Agent,
} from '@entities/agent';
import dynamic from 'next/dynamic';

// Lazy-loaded: the edit dialog embeds the JSON/YAML editor (js-yaml), only
// needed once the user opens it — keep it out of the initial route bundle.
const AgentEditDialog = dynamic(() => import('@features/agent-edit/ui/AgentEditDialog'));

function AgentDetailInner() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const search = useSearchParams();
  const { namespace } = useNamespace();
  const [tab, setTab] = useState(0);
  const [editOpen, setEditOpen] = useState(false);
  const [restartBusy, setRestartBusy] = useState(false);
  const [actionHandled, setActionHandled] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [deleteOpen, setDeleteOpen] = useState(false);

  const {
    data: agent,
    error: fetchError,
    isLoading: loading,
    mutate,
  } = useApi<Agent>(`/api/v1/namespaces/${namespace}/agents/${params.id}`);
  const fetchAgent = () => mutate();
  const error =
    actionError ??
    (fetchError instanceof Error
      ? fetchError.message
      : fetchError
        ? 'Failed to fetch agent'
        : null);

  const requestRestart = useCallback(async () => {
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
      // Seed the cache with the server response; no need to refetch.
      await mutate(updated, { revalidate: false });
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to set restart');
    } finally {
      setRestartBusy(false);
    }
  }, [agent, namespace, params.id, mutate]);

  const onDeleteAgent = useCallback(async () => {
    await deleteAgent(namespace, params.id);
    setDeleteOpen(false);
    router.push('/agents');
  }, [namespace, params.id, router]);

  // Honor ?action= once after the agent loads.
  useEffect(() => {
    if (!agent || actionHandled) return;
    const action = search.get('action');
    if (!action) return;
    setActionHandled(true);
    if (action === 'edit') {
      setEditOpen(true);
    } else if (action === 'restart') {
      void requestRestart();
    }
    router.replace(`/agents/${params.id}`);
  }, [agent, actionHandled, search, router, params.id, requestRestart]);

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
            <Button startIcon={<RestartIcon />} onClick={requestRestart} disabled={restartBusy}>
              Request restart
            </Button>
            <ReconcileButton
              kind="agent"
              namespace={namespace}
              name={agent.metadata.instanceUid}
              onReconciled={fetchAgent}
            />
            {!agent.status.connected && (
              <Button startIcon={<DeleteIcon />} color="error" onClick={() => setDeleteOpen(true)}>
                Delete
              </Button>
            )}
            <Button startIcon={<EditIcon />} variant="contained" onClick={() => setEditOpen(true)}>
              Edit spec
            </Button>
          </>
        }
      />

      <Stack direction="row" gap={1} alignItems="center" sx={{ mb: 2 }} flexWrap="wrap">
        <Typography variant="overline" color="text.secondary">
          Type
        </Typography>
        <Chip
          label={agentTypeLabel(agent.metadata.type)}
          color={isOtelCollector(agent.metadata.type) ? 'info' : 'default'}
          size="small"
          variant={isOtelCollector(agent.metadata.type) ? 'filled' : 'outlined'}
        />
      </Stack>

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
                Last reported: <TimeDisplay value={agent.status.lastReportedAt} />
              </Typography>
              <Typography variant="body2">Sequence #: {agent.status.sequenceNum ?? '—'}</Typography>
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
              {Object.entries(agent.status.effectiveConfig?.configMap.configMap ?? {}).length ===
              0 ? (
                <Typography color="text.secondary">No effective config reported.</Typography>
              ) : (
                Object.entries(agent.status.effectiveConfig?.configMap.configMap ?? {}).map(
                  ([name, file]) => (
                    <JsonBlock
                      key={name}
                      title={`${name} (${file.contentType})`}
                      value={file.body}
                    />
                  ),
                )
              )}
            </Stack>
          )}
          {tab === 2 && <JsonBlock value={agent.spec ?? {}} />}
          {tab === 3 && <JsonBlock value={agent.status.conditions ?? []} />}
          {tab === 4 && <JsonBlock value={agent} />}
        </CardContent>
      </Card>

      {editOpen && (
        <AgentEditDialog
          open
          agent={agent}
          onClose={() => setEditOpen(false)}
          onSaved={(saved) => {
            void mutate(saved, { revalidate: false });
            setEditOpen(false);
          }}
        />
      )}

      <ConfirmDialog
        open={deleteOpen}
        title="Delete agent"
        message={agentDeleteConfirmMessage(agent.metadata.instanceUid)}
        confirmLabel="Delete"
        destructive
        onClose={() => setDeleteOpen(false)}
        onConfirm={onDeleteAgent}
      />
    </Box>
  );
}

export default function AgentDetailPage() {
  return (
    <Suspense fallback={null}>
      <AgentDetailInner />
    </Suspense>
  );
}
