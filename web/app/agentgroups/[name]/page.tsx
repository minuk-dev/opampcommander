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
  Paper,
  Stack,
  Tab,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Tabs,
  Typography,
} from '@mui/material';
import {
  ArrowBack as ArrowBackIcon,
  Refresh as RefreshIcon,
  Edit as EditIcon,
} from '@mui/icons-material';
import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { useCallback, useEffect, useState } from 'react';
import PageHeader from '@/components/PageHeader';
import JsonBlock from '@/components/JsonBlock';
import { useNamespace } from '@/components/NamespaceProvider';
import { api } from '@/lib/api-client';
import type { Agent, AgentGroup, ListResponse } from '@/lib/types';
import AgentGroupEditDialog from '../AgentGroupEditDialog';

export default function AgentGroupDetailPage() {
  const params = useParams<{ name: string }>();
  const router = useRouter();
  const { namespace } = useNamespace();
  const [group, setGroup] = useState<AgentGroup | null>(null);
  const [members, setMembers] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [tab, setTab] = useState(0);
  const [editing, setEditing] = useState(false);

  const fetchAll = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [g, agents] = await Promise.all([
        api.get<AgentGroup>(
          `/api/v1/namespaces/${namespace}/agentgroups/${params.name}`,
        ),
        api.get<ListResponse<Agent>>(
          `/api/v1/namespaces/${namespace}/agentgroups/${params.name}/agents`,
          { query: { limit: 200 } },
        ),
      ]);
      setGroup(g);
      setMembers(agents.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch group');
    } finally {
      setLoading(false);
    }
  }, [namespace, params.name]);

  useEffect(() => {
    void fetchAll();
  }, [fetchAll]);

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
            <Button startIcon={<RefreshIcon />} onClick={fetchAll}>
              Refresh
            </Button>
            <Button startIcon={<EditIcon />} variant="contained" onClick={() => setEditing(true)}>
              Edit
            </Button>
          </>
        }
      />

      <Grid container spacing={2} sx={{ mb: 2 }}>
        {[
          ['Total', group.status.numAgents],
          ['Connected', group.status.numConnectedAgents],
          ['Healthy', group.status.numHealthyAgents],
          ['Unhealthy', group.status.numUnhealthyAgents],
          ['Not connected', group.status.numNotConnectedAgents],
        ].map(([label, value]) => (
          <Grid size={{ xs: 6, md: 2.4 }} key={label}>
            <Card>
              <CardContent>
                <Typography variant="overline" color="text.secondary">
                  {label}
                </Typography>
                <Typography variant="h5">{value}</Typography>
              </CardContent>
            </Card>
          </Grid>
        ))}
      </Grid>

      <Card>
        <Tabs value={tab} onChange={(_, v) => setTab(v)}>
          <Tab label={`Members (${members.length})`} />
          <Tab label="Selector" />
          <Tab label="Agent config" />
          <Tab label="Conditions" />
          <Tab label="Raw" />
        </Tabs>
        <Divider />
        <CardContent>
          {tab === 0 && (
            <TableContainer component={Paper} variant="outlined">
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>Instance UID</TableCell>
                    <TableCell>Connected</TableCell>
                    <TableCell>Healthy</TableCell>
                    <TableCell>Last reported</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {members.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={4} align="center">
                        No matching agents
                      </TableCell>
                    </TableRow>
                  ) : (
                    members.map((a) => (
                      <TableRow key={a.metadata.instanceUid} hover>
                        <TableCell sx={{ fontFamily: 'monospace' }}>
                          <Link href={`/agents/${a.metadata.instanceUid}`}>
                            {a.metadata.instanceUid}
                          </Link>
                        </TableCell>
                        <TableCell>
                          <Chip
                            label={a.status.connected ? 'Connected' : 'Disconnected'}
                            color={a.status.connected ? 'success' : 'default'}
                            size="small"
                          />
                        </TableCell>
                        <TableCell>
                          <Chip
                            label={a.status.componentHealth?.healthy ? 'Healthy' : 'Unhealthy'}
                            color={a.status.componentHealth?.healthy ? 'success' : 'warning'}
                            size="small"
                          />
                        </TableCell>
                        <TableCell>{a.status.lastReportedAt || '-'}</TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </TableContainer>
          )}
          {tab === 1 && (
            <Stack spacing={2}>
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
          {tab === 2 && <JsonBlock value={group.spec.agentConfig ?? {}} />}
          {tab === 3 && <JsonBlock value={group.status.conditions ?? []} />}
          {tab === 4 && <JsonBlock value={group} />}
        </CardContent>
      </Card>

      <AgentGroupEditDialog
        open={editing}
        mode="edit"
        initial={group}
        onClose={() => setEditing(false)}
        onSaved={() => {
          setEditing(false);
          void fetchAll();
        }}
      />
    </Box>
  );
}
