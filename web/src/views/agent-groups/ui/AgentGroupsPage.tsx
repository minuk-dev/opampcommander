'use client';

import { Eye, ListChecks, Pencil, Plus, RefreshCw, Trash2, Users } from 'lucide-react';
import Link from 'next/link';
import { useState } from 'react';
import {
  Alert,
  Badge,
  Button,
  ColumnPicker,
  ConfirmDialog,
  Label,
  ListFilterBar,
  PageHeader,
  PaginationFooter,
  RowActionsMenu,
  Spinner,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeaderCell,
  TableRow,
  TableWrap,
  Tooltip,
} from '@shared/ui';
import { TimeDisplay } from '@shared/preferences';
import { useNamespace } from '@entities/namespace';
import { api } from '@shared/api';
import {
  cn,
  EMPTY_LIST_FILTERS,
  hasListFilters,
  listFilterQuery,
  useColumnVisibility,
  useCursorPagination,
  type ColumnConfig,
  type ListFilters,
} from '@shared/lib';
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
  const [filters, setFilters] = useState<ListFilters>(EMPTY_LIST_FILTERS);

  const { visible, isVisible, toggle } = useColumnVisibility('agentgroups', AGENT_GROUP_COLUMNS);
  // +1 for the always-present Actions column.
  const colSpan = AGENT_GROUP_COLUMNS.filter((c) => isVisible(c.id)).length + 1;

  // A group's labels are its metadata.attributes, and its name is filtered by
  // prefix — both answered by the datastore, so the paginated total stays in
  // step with the rows.
  const pagination = useCursorPagination<AgentGroup>(
    `/api/v1/namespaces/${namespace}/agentgroups`,
    { query: listFilterQuery(filters) },
  );
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
    <div>
      <PageHeader
        title="Agent Groups"
        subtitle={`Namespace: ${namespace}`}
        actions={
          <>
            <Label className="flex cursor-pointer items-center gap-1.5">
              <Switch checked={showDisconnected} onCheckedChange={setShowDisconnected} />
              Show disconnected
            </Label>
            <Button
              variant="ghost"
              size="icon-sm"
              aria-label="Refresh"
              onClick={() => void fetchGroups()}
            >
              <RefreshCw className={cn(loading && 'animate-spin')} aria-hidden />
            </Button>
            <ColumnPicker columns={AGENT_GROUP_COLUMNS} visible={visible} onToggle={toggle} />
            <Button size="sm" onClick={() => setCreateOpen(true)}>
              <Plus aria-hidden />
              New group
            </Button>
          </>
        }
      />

      <ListFilterBar value={filters} onChange={setFilters} />

      {error && (
        <Alert severity="error" className="mb-3">
          {error}
        </Alert>
      )}

      <TableWrap>
        <Table>
          <TableHead>
            <TableRow className="hover:bg-transparent">
              {isVisible('name') && <TableHeaderCell>Name</TableHeaderCell>}
              {isVisible('priority') && <TableHeaderCell>Priority</TableHeaderCell>}
              {isVisible('agents') && <TableHeaderCell>Agents</TableHeaderCell>}
              {isVisible('connected') && <TableHeaderCell>Connected</TableHeaderCell>}
              {isVisible('healthy') && <TableHeaderCell>Healthy</TableHeaderCell>}
              {isVisible('created') && <TableHeaderCell>Created</TableHeaderCell>}
              <TableHeaderCell className="w-10 text-right">Actions</TableHeaderCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {loading ? (
              <TableRow className="hover:bg-transparent">
                <TableCell colSpan={colSpan} className="py-8">
                  <Spinner className="mx-auto size-5" />
                </TableCell>
              </TableRow>
            ) : groups.length === 0 ? (
              <TableRow className="hover:bg-transparent">
                <TableCell colSpan={colSpan} className="py-8 text-center text-muted-foreground">
                  {hasListFilters(filters)
                    ? 'No agent groups match the filters'
                    : 'No agent groups'}
                </TableCell>
              </TableRow>
            ) : (
              groups.map((g) => (
                <TableRow key={g.metadata.name}>
                  {isVisible('name') && (
                    <TableCell>
                      <Tooltip content="View agents in this group" side="right">
                        <Link
                          href={`/agents?agentGroup=${encodeURIComponent(g.metadata.name)}`}
                          className="font-medium text-primary hover:underline"
                        >
                          {g.metadata.name}
                        </Link>
                      </Tooltip>
                    </TableCell>
                  )}
                  {isVisible('priority') && (
                    <TableCell className="tnum">{g.spec.priority}</TableCell>
                  )}
                  {isVisible('agents') && (
                    <TableCell>
                      <Tooltip
                        content={
                          showDisconnected
                            ? 'All agents in this group (connected + disconnected)'
                            : 'Connected agents in this group'
                        }
                      >
                        <Link href={`/agents?agentGroup=${encodeURIComponent(g.metadata.name)}`}>
                          <Badge className="tnum hover:bg-accent">
                            {showDisconnected ? g.status.numAgents : g.status.numConnectedAgents}
                          </Badge>
                        </Link>
                      </Tooltip>
                    </TableCell>
                  )}
                  {isVisible('connected') && (
                    <TableCell>
                      <Badge variant="success" className="tnum">
                        {g.status.numConnectedAgents}/{g.status.numAgents}
                      </Badge>
                    </TableCell>
                  )}
                  {isVisible('healthy') && (
                    <TableCell>
                      <Badge
                        variant={g.status.numUnhealthyAgents ? 'warning' : 'success'}
                        className="tnum"
                      >
                        {g.status.numHealthyAgents}/{g.status.numAgents}
                      </Badge>
                    </TableCell>
                  )}
                  {isVisible('created') && (
                    <TableCell>
                      <TimeDisplay value={g.metadata.createdAt} />
                    </TableCell>
                  )}
                  <TableCell className="text-right">
                    <RowActionsMenu
                      actions={[
                        {
                          label: 'View detail',
                          icon: <Eye aria-hidden />,
                          href: `/agentgroups/${g.metadata.name}`,
                        },
                        {
                          label: 'View agents',
                          icon: <Users aria-hidden />,
                          href: `/agents?agentGroup=${encodeURIComponent(g.metadata.name)}`,
                        },
                        {
                          label: 'Edit',
                          icon: <Pencil aria-hidden />,
                          href: `/agentgroups/${g.metadata.name}?action=edit`,
                        },
                        {
                          label: 'Apply remote config',
                          icon: <ListChecks aria-hidden />,
                          href: `/agentgroups/${g.metadata.name}?action=apply`,
                        },
                        {
                          label: 'Delete',
                          icon: <Trash2 aria-hidden />,
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
      </TableWrap>

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
    </div>
  );
}
