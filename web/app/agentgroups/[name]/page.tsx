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
  Tooltip,
  Typography,
} from '@mui/material';
import {
  ArrowBack as ArrowBackIcon,
  Refresh as RefreshIcon,
  Edit as EditIcon,
  PeopleAlt as PeopleAltIcon,
  CalendarToday as CalendarIcon,
} from '@mui/icons-material';
import Link from 'next/link';
import { useParams, useRouter, useSearchParams } from 'next/navigation';
import { Suspense, useEffect, useState } from 'react';
import PageHeader from '@/components/PageHeader';
import JsonBlock from '@/components/JsonBlock';
import TimeDisplay from '@/components/TimeDisplay';
import ConfirmDialog from '@/components/ConfirmDialog';
import { useNamespace } from '@/components/NamespaceProvider';
import { api } from '@/lib/api-client';
import { useApi } from '@/lib/swr';
import type { AgentGroup } from '@/lib/types';
import AgentGroupEditDialog from '../AgentGroupEditDialog';

function AgentGroupDetailInner() {
  const params = useParams<{ name: string }>();
  const router = useRouter();
  const search = useSearchParams();
  const { namespace } = useNamespace();
  const [tab, setTab] = useState(0);
  const [editing, setEditing] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [actionHandled, setActionHandled] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);

  const {
    data: group,
    error: fetchError,
    isLoading: loading,
    mutate,
  } = useApi<AgentGroup>(`/api/v1/namespaces/${namespace}/agentgroups/${params.name}`);
  const fetchGroup = () => mutate();
  const error =
    actionError ??
    (fetchError instanceof Error
      ? fetchError.message
      : fetchError
        ? 'Failed to fetch group'
        : null);

  // Auto-trigger ?action= once after load (e.g. from the list page menu).
  useEffect(() => {
    if (!group || actionHandled) return;
    const action = search.get('action');
    if (!action) return;
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setActionHandled(true);
    if (action === 'edit') {
      setEditing(true);
    } else if (action === 'delete') {
      setDeleting(true);
    }
    router.replace(`/agentgroups/${params.name}`);
  }, [group, actionHandled, search, router, params.name]);

  const onDelete = async () => {
    try {
      await api.delete(`/api/v1/namespaces/${namespace}/agentgroups/${params.name}`);
      router.push('/agentgroups');
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to delete');
      setDeleting(false);
    }
  };

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" mt={6}>
        <CircularProgress />
      </Box>
    );
  }
  if (error || !group) {
    return (
      <Box>
        <Button startIcon={<ArrowBackIcon />} onClick={() => router.back()} sx={{ mb: 2 }}>
          Back
        </Button>
        <Alert severity="error">{error || 'Group not found'}</Alert>
      </Box>
    );
  }

  const agentsHref = `/agents?agentGroup=${encodeURIComponent(group.metadata.name)}`;

  return (
    <Box>
      <Button
        startIcon={<ArrowBackIcon />}
        onClick={() => router.push('/agentgroups')}
        sx={{ mb: 2 }}
      >
        Back to groups
      </Button>
      <PageHeader
        title={group.metadata.name}
        subtitle={`Namespace: ${group.metadata.namespace} · priority ${group.spec.priority}`}
        actions={
          <>
            <Button startIcon={<RefreshIcon />} onClick={fetchGroup}>
              Refresh
            </Button>
            <Button
              startIcon={<PeopleAltIcon />}
              component={Link}
              href={agentsHref}
              variant="outlined"
            >
              View agents
            </Button>
            <Button startIcon={<EditIcon />} variant="contained" onClick={() => setEditing(true)}>
              Edit
            </Button>
          </>
        }
      />

      <Grid container spacing={2} sx={{ mb: 2 }}>
        {[
          ['Total', group.status.numAgents, 'default' as const],
          ['Connected', group.status.numConnectedAgents, 'success' as const],
          ['Healthy', group.status.numHealthyAgents, 'success' as const],
          ['Unhealthy', group.status.numUnhealthyAgents, 'warning' as const],
          ['Not connected', group.status.numNotConnectedAgents, 'default' as const],
        ].map(([label, value, color]) => (
          <Grid size={{ xs: 6, md: 2.4 }} key={String(label)}>
            <Tooltip title={`View agents in ${group.metadata.name}`} placement="top">
              <Card
                component={Link}
                href={agentsHref}
                sx={{
                  display: 'block',
                  textDecoration: 'none',
                  color: 'inherit',
                  cursor: 'pointer',
                  transition: 'transform 0.1s, box-shadow 0.1s',
                  '&:hover': { transform: 'translateY(-1px)', boxShadow: 4 },
                }}
              >
                <CardContent>
                  <Typography variant="overline" color="text.secondary">
                    {label}
                  </Typography>
                  <Typography
                    variant="h5"
                    color={
                      color === 'warning' && Number(value) > 0
                        ? 'warning.main'
                        : color === 'success' && Number(value) > 0
                          ? 'success.main'
                          : 'text.primary'
                    }
                  >
                    {value}
                  </Typography>
                </CardContent>
              </Card>
            </Tooltip>
          </Grid>
        ))}
      </Grid>

      <Card sx={{ mb: 2 }}>
        <CardContent>
          <Stack
            direction={{ xs: 'column', md: 'row' }}
            spacing={2}
            divider={<Divider orientation="vertical" flexItem />}
          >
            <Stack direction="row" spacing={1} alignItems="center">
              <CalendarIcon fontSize="small" color="action" />
              <Box>
                <Typography variant="caption" color="text.secondary">
                  Created
                </Typography>
                <Typography variant="body2" component="div">
                  <TimeDisplay value={group.metadata.createdAt} />
                </Typography>
              </Box>
            </Stack>
            {group.metadata.deletedAt && (
              <Box>
                <Typography variant="caption" color="text.secondary">
                  Deleted
                </Typography>
                <Typography variant="body2" component="div">
                  <TimeDisplay value={group.metadata.deletedAt} />
                </Typography>
              </Box>
            )}
            <Box>
              <Typography variant="caption" color="text.secondary">
                Attributes
              </Typography>
              <Stack direction="row" gap={0.5} flexWrap="wrap">
                {Object.entries(group.metadata.attributes || {}).length === 0 ? (
                  <Typography variant="body2" color="text.secondary">
                    none
                  </Typography>
                ) : (
                  Object.entries(group.metadata.attributes).map(([k, v]) => (
                    <Chip key={k} label={`${k}=${v}`} size="small" variant="outlined" />
                  ))
                )}
              </Stack>
            </Box>
          </Stack>
        </CardContent>
      </Card>

      <Card>
        <Tabs value={tab} onChange={(_, v) => setTab(v)}>
          <Tab label="Selector" />
          <Tab label="Agent config" />
          <Tab label="Conditions" />
          <Tab label="Raw" />
        </Tabs>
        <Divider />
        <CardContent>
          {tab === 0 && (
            <Stack spacing={2}>
              <Typography variant="body2" color="text.secondary">
                Agents are matched to this group when their identifying / non-identifying attributes
                contain all of the keys/values defined below.
              </Typography>
              <JsonBlock
                title="Identifying attributes"
                value={group.spec.selector.identifyingAttributes ?? {}}
              />
              <JsonBlock
                title="Non-identifying attributes"
                value={group.spec.selector.nonIdentifyingAttributes ?? {}}
              />
            </Stack>
          )}
          {tab === 1 && <JsonBlock value={group.spec.agentConfig ?? {}} />}
          {tab === 2 && <JsonBlock value={group.status.conditions ?? []} />}
          {tab === 3 && <JsonBlock value={group} />}
        </CardContent>
      </Card>

      <AgentGroupEditDialog
        open={editing}
        mode="edit"
        initial={group}
        onClose={() => setEditing(false)}
        onSaved={() => {
          setEditing(false);
          void fetchGroup();
        }}
      />
      <ConfirmDialog
        open={deleting}
        title="Delete agent group"
        message={`Delete "${group.metadata.name}"? This cannot be undone.`}
        confirmLabel="Delete"
        destructive
        onClose={() => setDeleting(false)}
        onConfirm={onDelete}
      />
    </Box>
  );
}

export default function AgentGroupDetailPage() {
  return (
    <Suspense fallback={null}>
      <AgentGroupDetailInner />
    </Suspense>
  );
}
