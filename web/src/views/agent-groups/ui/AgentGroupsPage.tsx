'use client';

import {
  Alert,
  Box,
  Button,
  Chip,
  CircularProgress,
  FormControlLabel,
  IconButton,
  Paper,
  Switch,
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
import {
  PageHeader,
  ConfirmDialog,
  PaginationFooter,
  RowActionsMenu,
  ColumnPicker,
} from '@shared/ui';
import { TimeDisplay } from '@shared/preferences';
import { useNamespace } from '@entities/namespace';
import { api } from '@shared/api';
import { type ColumnConfig, useColumnVisibility, useCursorPagination } from '@shared/lib';
import dynamic from 'next/dynamic';
import type { AgentGroup } from '@entities/agent-group';

// Lazy-loaded: the edit dialog embeds the JSON/YAML editor (js-yaml), only
// needed once the user opens it — keep it out of the initial route bundle.
const AgentGroupEditDialog = dynamic(
  () => import('@features/agent-group-edit/ui/AgentGroupEditDialog'),
);

// Columns for the agent groups table. `name` is locked (the row identifier);
// the rest are toggleable via the column picker and persisted per user.
const AGENT_GROUP_COLUMNS: ColumnConfig[] = [
  { id: 'name', label: 'Name', locked: true },
  { id: 'priority', label: 'Priority' },
  { id: 'agents', label: 'Agents' },
  { id: 'connected', label: 'Connected' },
  { id: 'healthy', label: 'Healthy' },
  { id: 'created', label: 'Created' },
];

export default function AgentGroupsPage() {
  const { namespace } = useNamespace();
  const [createOpen, setCreateOpen] = useState(false);
  const [editing, setEditing] = useState<AgentGroup | null>(null);
  const [deleting, setDeleting] = useState<AgentGroup | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  // The "Agents" count shows connected agents only by default so it agrees with
  // the agents list (which also hides disconnected by default); the toggle
  // switches it to the full membership count.
  const [showDisconnected, setShowDisconnected] = useState(false);

  const { visible, isVisible, toggle } = useColumnVisibility('agentgroups', AGENT_GROUP_COLUMNS);
  // +1 for the always-present Actions column.
  const colSpan = AGENT_GROUP_COLUMNS.filter((c) => isVisible(c.id)).length + 1;

  const pagination = useCursorPagination<AgentGroup>(`/api/v1/namespaces/${namespace}/agentgroups`);
  const { items: groups, isLoading: loading, error: fetchError, refresh } = pagination;
  const error =
    actionError ??
    (fetchError instanceof Error
      ? fetchError.message
      : fetchError
        ? 'Failed to fetch groups'
        : null);

  const fetchGroups = () => refresh();

  const onDelete = async () => {
    if (!deleting) return;
    try {
      await api.delete(`/api/v1/namespaces/${namespace}/agentgroups/${deleting.metadata.name}`);
      setDeleting(null);
      setActionError(null);
      refresh();
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
            <FormControlLabel
              control={
                <Switch
                  size="small"
                  checked={showDisconnected}
                  onChange={(e) => setShowDisconnected(e.target.checked)}
                />
              }
              label="Show disconnected"
              sx={{ color: 'text.secondary', mr: 1 }}
            />
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

      <Box sx={{ display: 'flex', justifyContent: 'flex-end', mb: 1 }}>
        <ColumnPicker columns={AGENT_GROUP_COLUMNS} visible={visible} onToggle={toggle} />
      </Box>

      <TableContainer component={Paper}>
        <Table>
          <TableHead>
            <TableRow>
              {isVisible('name') && <TableCell>Name</TableCell>}
              {isVisible('priority') && <TableCell>Priority</TableCell>}
              {isVisible('agents') && <TableCell>Agents</TableCell>}
              {isVisible('connected') && <TableCell>Connected</TableCell>}
              {isVisible('healthy') && <TableCell>Healthy</TableCell>}
              {isVisible('created') && <TableCell>Created</TableCell>}
              <TableCell align="right">Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {loading ? (
              <TableRow>
                <TableCell colSpan={colSpan} align="center">
                  <CircularProgress size={24} />
                </TableCell>
              </TableRow>
            ) : groups.length === 0 ? (
              <TableRow>
                <TableCell colSpan={colSpan} align="center">
                  No agent groups
                </TableCell>
              </TableRow>
            ) : (
              groups.map((g) => (
                <TableRow key={g.metadata.name} hover>
                  {isVisible('name') && (
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
                  )}
                  {isVisible('priority') && <TableCell>{g.spec.priority}</TableCell>}
                  {isVisible('agents') && (
                    <TableCell>
                      <Tooltip
                        title={
                          showDisconnected
                            ? 'All agents in this group (connected + disconnected)'
                            : 'Connected agents in this group'
                        }
                        placement="top"
                      >
                        <Chip
                          component={Link}
                          href={`/agents?agentGroup=${encodeURIComponent(g.metadata.name)}`}
                          label={
                            showDisconnected ? g.status.numAgents : g.status.numConnectedAgents
                          }
                          size="small"
                          clickable
                        />
                      </Tooltip>
                    </TableCell>
                  )}
                  {isVisible('connected') && (
                    <TableCell>
                      <Chip
                        label={`${g.status.numConnectedAgents}/${g.status.numAgents}`}
                        color="success"
                        size="small"
                        variant="outlined"
                      />
                    </TableCell>
                  )}
                  {isVisible('healthy') && (
                    <TableCell>
                      <Chip
                        label={`${g.status.numHealthyAgents}/${g.status.numAgents}`}
                        color={g.status.numUnhealthyAgents ? 'warning' : 'success'}
                        size="small"
                        variant="outlined"
                      />
                    </TableCell>
                  )}
                  {isVisible('created') && (
                    <TableCell>
                      <TimeDisplay value={g.metadata.createdAt} />
                    </TableCell>
                  )}
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

      <PaginationFooter pagination={pagination} />

      {createOpen && (
        <AgentGroupEditDialog
          open
          mode="create"
          onClose={() => setCreateOpen(false)}
          onSaved={() => {
            setCreateOpen(false);
            void fetchGroups();
          }}
        />
      )}
      {editing !== null && (
        <AgentGroupEditDialog
          open
          mode="edit"
          initial={editing ?? undefined}
          onClose={() => setEditing(null)}
          onSaved={() => {
            setEditing(null);
            void fetchGroups();
          }}
        />
      )}
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
