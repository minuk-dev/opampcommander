'use client';

import {
  Alert,
  Box,
  Button,
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
} from '@mui/material';
import {
  Refresh as RefreshIcon,
  Add as AddIcon,
  Delete as DeleteIcon,
  Edit as EditIcon,
  PeopleAlt as PeopleAltIcon,
  Visibility as ViewIcon,
} from '@mui/icons-material';
import Link from 'next/link';
import { Tooltip } from '@mui/material';
import { useState } from 'react';
import PageHeader from '@/components/PageHeader';
import ConfirmDialog from '@/components/ConfirmDialog';
import RowActionsMenu from '@/components/RowActionsMenu';
import { useNamespace } from '@/components/NamespaceProvider';
import { api } from '@/lib/api-client';
import { useApi } from '@/lib/swr';
import type { AgentGroup, ListResponse } from '@/lib/types';
import AgentGroupEditDialog from './AgentGroupEditDialog';

export default function AgentGroupsPage() {
  const { namespace } = useNamespace();
  const [createOpen, setCreateOpen] = useState(false);
  const [editing, setEditing] = useState<AgentGroup | null>(null);
  const [deleting, setDeleting] = useState<AgentGroup | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  const {
    data,
    error: fetchError,
    isLoading,
    mutate,
  } = useApi<ListResponse<AgentGroup>>([
    `/api/v1/namespaces/${namespace}/agentgroups`,
    { limit: 200 },
  ]);
  const groups = data?.items ?? [];
  const loading = isLoading;
  const error =
    actionError ??
    (fetchError instanceof Error
      ? fetchError.message
      : fetchError
        ? 'Failed to fetch groups'
        : null);

  const fetchGroups = () => mutate();

  const onDelete = async () => {
    if (!deleting) return;
    try {
      await api.delete(`/api/v1/namespaces/${namespace}/agentgroups/${deleting.metadata.name}`);
      setDeleting(null);
      setActionError(null);
      await mutate();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to delete');
    }
  };

  return (
    <Box>
      <PageHeader
        title="Agent Groups"
        subtitle={`Namespace: ${namespace}`}
        actions={
          <>
            <IconButton color="primary" onClick={fetchGroups}>
              <RefreshIcon />
            </IconButton>
            <Button startIcon={<AddIcon />} variant="contained" onClick={() => setCreateOpen(true)}>
              New group
            </Button>
          </>
        }
      />

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      <TableContainer component={Paper}>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>Name</TableCell>
              <TableCell>Priority</TableCell>
              <TableCell>Agents</TableCell>
              <TableCell>Connected</TableCell>
              <TableCell>Healthy</TableCell>
              <TableCell>Created</TableCell>
              <TableCell align="right">Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {loading ? (
              <TableRow>
                <TableCell colSpan={7} align="center">
                  <CircularProgress size={24} />
                </TableCell>
              </TableRow>
            ) : groups.length === 0 ? (
              <TableRow>
                <TableCell colSpan={7} align="center">
                  No agent groups
                </TableCell>
              </TableRow>
            ) : (
              groups.map((g) => (
                <TableRow key={g.metadata.name} hover>
                  <TableCell>
                    <Tooltip title="View agents in this group" placement="right">
                      <Link
                        href={`/agents?agentGroup=${encodeURIComponent(g.metadata.name)}`}
                        style={{ fontWeight: 500 }}
                      >
                        {g.metadata.name}
                      </Link>
                    </Tooltip>
                  </TableCell>
                  <TableCell>{g.spec.priority}</TableCell>
                  <TableCell>
                    <Tooltip title="View agents in this group" placement="top">
                      <Chip
                        component={Link}
                        href={`/agents?agentGroup=${encodeURIComponent(g.metadata.name)}`}
                        label={g.status.numAgents}
                        size="small"
                        clickable
                      />
                    </Tooltip>
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={`${g.status.numConnectedAgents}/${g.status.numAgents}`}
                      color="success"
                      size="small"
                      variant="outlined"
                    />
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={`${g.status.numHealthyAgents}/${g.status.numAgents}`}
                      color={g.status.numUnhealthyAgents ? 'warning' : 'success'}
                      size="small"
                      variant="outlined"
                    />
                  </TableCell>
                  <TableCell>{g.metadata.createdAt}</TableCell>
                  <TableCell align="right">
                    <RowActionsMenu
                      actions={[
                        {
                          label: 'View detail',
                          icon: <ViewIcon fontSize="small" />,
                          href: `/agentgroups/${g.metadata.name}`,
                        },
                        {
                          label: 'View agents',
                          icon: <PeopleAltIcon fontSize="small" />,
                          href: `/agents?agentGroup=${encodeURIComponent(g.metadata.name)}`,
                        },
                        {
                          label: 'Edit',
                          icon: <EditIcon fontSize="small" />,
                          href: `/agentgroups/${g.metadata.name}?action=edit`,
                        },
                        {
                          label: 'Delete',
                          icon: <DeleteIcon fontSize="small" />,
                          destructive: true,
                          divider: true,
                          onClick: () => setDeleting(g),
                        },
                      ]}
                    />
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>

      <AgentGroupEditDialog
        open={createOpen}
        mode="create"
        onClose={() => setCreateOpen(false)}
        onSaved={() => {
          setCreateOpen(false);
          void fetchGroups();
        }}
      />
      <AgentGroupEditDialog
        open={editing !== null}
        mode="edit"
        initial={editing ?? undefined}
        onClose={() => setEditing(null)}
        onSaved={() => {
          setEditing(null);
          void fetchGroups();
        }}
      />
      <ConfirmDialog
        open={deleting !== null}
        title="Delete agent group"
        message={`Are you sure you want to delete "${deleting?.metadata.name}"?`}
        confirmLabel="Delete"
        destructive
        onClose={() => setDeleting(null)}
        onConfirm={onDelete}
      />
    </Box>
  );
}
