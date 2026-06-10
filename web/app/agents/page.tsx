'use client';

import {
  Alert,
  Autocomplete,
  Box,
  Button,
  Chip,
  CircularProgress,
  FormControl,
  FormControlLabel,
  IconButton,
  InputLabel,
  MenuItem,
  Paper,
  Select,
  Stack,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Tooltip,
} from '@mui/material';
import {
  Refresh as RefreshIcon,
  Search as SearchIcon,
  Group as GroupIcon,
  Visibility as ViewIcon,
  Edit as EditIcon,
  RestartAlt as RestartIcon,
  Delete as DeleteIcon,
} from '@mui/icons-material';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { Suspense, useEffect, useState } from 'react';
import PageHeader from '@/components/PageHeader';
import PaginationFooter from '@/components/PaginationFooter';
import RowActionsMenu, { type RowAction } from '@/components/RowActionsMenu';
import ConfirmDialog from '@/components/ConfirmDialog';
import ColumnPicker from '@/components/ColumnPicker';
import TimeDisplay from '@/components/TimeDisplay';
import { useNamespace } from '@/components/NamespaceProvider';
import { agentDeleteConfirmMessage, capabilityNames, deleteAgent } from '@/lib/agents';
import { type ColumnConfig, useColumnVisibility } from '@/lib/column-visibility';
import { useCursorPagination } from '@/lib/pagination';
import { useApi } from '@/lib/swr';
import type { Agent, AgentGroup, ListResponse } from '@/lib/types';

type SearchMode = 'uid' | 'group' | 'description';

// `lcNeedle` is expected to be pre-lowercased by the caller so we don't redo
// it for every agent in a filter pass.
function attrMatchesDescription(agent: Agent, lcNeedle: string): boolean {
  const desc = agent.metadata.description;
  const collect = [
    ...Object.entries(desc?.identifyingAttributes ?? {}),
    ...Object.entries(desc?.nonIdentifyingAttributes ?? {}),
  ];
  return collect.some(
    ([k, v]) => k.toLowerCase().includes(lcNeedle) || v.toLowerCase().includes(lcNeedle),
  );
}

// Columns for the agents table. `instanceUid` is locked (the row identifier);
// `sequence` and the verbose capability/attribute columns are off by default so
// the table stays readable, and users opt into them via the column picker.
const AGENT_COLUMNS: ColumnConfig[] = [
  { id: 'instanceUid', label: 'Instance UID', locked: true },
  { id: 'connected', label: 'Connected' },
  { id: 'healthy', label: 'Healthy' },
  { id: 'type', label: 'Type' },
  { id: 'lastReported', label: 'Last Reported' },
  { id: 'sequence', label: 'Sequence', defaultVisible: false },
  { id: 'capabilities', label: 'Capabilities', defaultVisible: false },
  {
    id: 'identifyingAttributes',
    label: 'Description (identifying attributes)',
    defaultVisible: false,
  },
  {
    id: 'nonIdentifyingAttributes',
    label: 'Description (non-identifying attributes)',
    defaultVisible: false,
  },
];

// Render an attribute map as compact key=value chips for a table cell.
function AttrChips({ attrs }: { attrs: Record<string, string> | undefined }) {
  const entries = Object.entries(attrs ?? {});
  if (entries.length === 0) return <>-</>;
  return (
    <Stack direction="row" gap={0.5} flexWrap="wrap" sx={{ maxWidth: 320 }}>
      {entries.map(([k, v]) => (
        <Chip key={k} label={`${k}=${v}`} size="small" variant="outlined" />
      ))}
    </Stack>
  );
}

function AgentsInner() {
  const router = useRouter();
  const search = useSearchParams();
  const { namespace } = useNamespace();

  const agentGroupParam = search.get('agentGroup') || '';
  const qParam = search.get('q') || '';
  const descParam = search.get('desc') || '';
  const modeParam =
    (search.get('mode') as SearchMode | null) ||
    (agentGroupParam ? 'group' : descParam ? 'description' : 'uid');

  const [mode, setMode] = useState<SearchMode>(modeParam);
  const [query, setQuery] = useState(qParam);
  const [deleting, setDeleting] = useState<Agent | null>(null);
  // Disconnected agents are hidden by default; the toggle reveals them. When
  // hidden we pass connected=true so the server filters them out (keeping the
  // paginated total accurate) rather than filtering the current page client-side.
  const [showDisconnected, setShowDisconnected] = useState(false);

  const { visible, isVisible, toggle } = useColumnVisibility('agents', AGENT_COLUMNS);
  // +1 for the always-present Actions column.
  const colSpan = AGENT_COLUMNS.filter((c) => isVisible(c.id)).length + 1;

  // The three search modes hit different endpoints; description mode reuses the
  // plain list and filters client-side (see visibleAgents below).
  let listPath: string;
  const listQuery: Record<string, string> = {};
  if (agentGroupParam) {
    listPath = `/api/v1/namespaces/${namespace}/agentgroups/${agentGroupParam}/agents`;
  } else if (qParam) {
    listPath = `/api/v1/namespaces/${namespace}/agents/search`;
    listQuery.q = qParam;
  } else {
    listPath = `/api/v1/namespaces/${namespace}/agents`;
  }
  if (!showDisconnected) {
    listQuery.connected = 'true';
  }

  const pagination = useCursorPagination<Agent>(listPath, { query: listQuery });
  const { items: agents, isLoading: loading, error: fetchError } = pagination;
  const error =
    fetchError instanceof Error ? fetchError.message : fetchError ? 'Failed to fetch agents' : null;

  // Group list for the "Group" autocomplete + chip cross-reference. SWR dedupes
  // this with the same fetch on other pages and silently no-ops on RBAC denial.
  const { data: groupData } = useApi<ListResponse<AgentGroup>>([
    `/api/v1/namespaces/${namespace}/agentgroups`,
    { limit: 500 },
  ]);
  const groupOptions = groupData?.items ?? [];

  // Keep the local input synced when URL drives the value (mode change etc.)
  useEffect(() => {
    if (mode === 'uid') setQuery(qParam);
    else if (mode === 'description') setQuery(descParam);
    else if (mode === 'group') setQuery(agentGroupParam);
  }, [mode, qParam, descParam, agentGroupParam]);

  // Sync mode state if URL changes externally
  useEffect(() => {
    setMode(modeParam);
  }, [modeParam]);

  const updateUrl = (next: {
    q?: string;
    agentGroup?: string;
    desc?: string;
    mode?: SearchMode;
  }) => {
    const params = new URLSearchParams();
    const m = next.mode ?? mode;
    const q = next.q ?? (m === 'uid' ? qParam : '');
    const g = next.agentGroup ?? (m === 'group' ? agentGroupParam : '');
    const d = next.desc ?? (m === 'description' ? descParam : '');
    if (q) params.set('q', q);
    if (g) params.set('agentGroup', g);
    if (d) params.set('desc', d);
    if (m !== 'uid' || !q) params.set('mode', m);
    const qs = params.toString();
    router.replace(qs ? `/agents?${qs}` : '/agents');
  };

  const onSearch = (e: React.FormEvent) => {
    e.preventDefault();
    const value = query.trim();
    if (mode === 'uid') {
      updateUrl({ q: value, agentGroup: '', desc: '', mode: 'uid' });
    } else if (mode === 'group') {
      updateUrl({ agentGroup: value, q: '', desc: '', mode: 'group' });
    } else {
      updateUrl({ desc: value, q: '', agentGroup: '', mode: 'description' });
    }
  };

  const setSearchMode = (next: SearchMode) => {
    setMode(next);
    setQuery('');
    updateUrl({ q: '', agentGroup: '', desc: '', mode: next });
  };

  const clearGroup = () => updateUrl({ agentGroup: '' });
  const clearSearch = () => {
    setQuery('');
    updateUrl({ q: '' });
  };

  // Only disconnected agents can be deleted; a connected one would just be
  // recreated on its next report (the server enforces this with a 409 too).
  // reset() (not refresh()) revalidates from page 0 so deleting the last row on a
  // later page doesn't strand the user on a now-empty cursor page.
  const onDelete = async () => {
    if (!deleting) return;
    await deleteAgent(namespace, deleting.metadata.instanceUid);
    setDeleting(null);
    pagination.reset();
  };

  const filterActive = Boolean(agentGroupParam || qParam || descParam);
  const lcDesc = descParam.toLowerCase();
  const visibleAgents = descParam
    ? agents.filter((a) => attrMatchesDescription(a, lcDesc))
    : agents;

  return (
    <Box>
      <PageHeader
        title="Agents"
        subtitle={`Namespace: ${namespace}`}
        actions={
          <IconButton color="primary" onClick={() => pagination.refresh()}>
            <RefreshIcon />
          </IconButton>
        }
      />

      <Paper sx={{ p: 2, mb: 2 }}>
        <Stack spacing={1.5}>
          <form onSubmit={onSearch}>
            <Stack direction={{ xs: 'column', sm: 'row' }} gap={1}>
              <FormControl size="small" sx={{ minWidth: 180 }}>
                <InputLabel id="search-mode-label">Search by</InputLabel>
                <Select
                  labelId="search-mode-label"
                  label="Search by"
                  value={mode}
                  onChange={(e) => setSearchMode(e.target.value as SearchMode)}
                >
                  <MenuItem value="group">Agent Group</MenuItem>
                  <MenuItem value="uid">Instance UID</MenuItem>
                  <MenuItem value="description">Description</MenuItem>
                </Select>
              </FormControl>

              {mode === 'group' ? (
                <Autocomplete
                  size="small"
                  fullWidth
                  freeSolo
                  options={groupOptions.map((g) => g.metadata.name)}
                  value={query}
                  onInputChange={(_, v) => setQuery(v)}
                  onChange={(_, v) =>
                    updateUrl({
                      agentGroup: v ?? '',
                      q: '',
                      desc: '',
                      mode: 'group',
                    })
                  }
                  renderInput={(params) => (
                    <TextField
                      {...params}
                      placeholder="Select or type an agent group name…"
                      InputProps={{
                        ...params.InputProps,
                        startAdornment: (
                          <>
                            <GroupIcon sx={{ ml: 1, mr: 0.5, color: 'text.secondary' }} />
                            {params.InputProps.startAdornment}
                          </>
                        ),
                      }}
                    />
                  )}
                />
              ) : (
                <TextField
                  size="small"
                  fullWidth
                  placeholder={
                    mode === 'uid'
                      ? 'Instance UID contains… (server-side)'
                      : 'Attribute key/value contains… (client-side, current page)'
                  }
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  InputProps={{
                    startAdornment: <SearchIcon sx={{ mr: 1, color: 'text.secondary' }} />,
                  }}
                />
              )}

              <Button type="submit" variant="contained">
                Search
              </Button>
            </Stack>
          </form>

          <FormControlLabel
            control={
              <Switch
                size="small"
                checked={showDisconnected}
                onChange={(e) => setShowDisconnected(e.target.checked)}
              />
            }
            label="Show disconnected agents"
            sx={{ color: 'text.secondary' }}
          />

          {mode === 'description' && (
            <Box sx={{ color: 'text.secondary', fontSize: 12 }}>
              Description search filters the current page on the client. Combine with a smaller
              namespace or refine via UID / group for large deployments.
            </Box>
          )}

          {filterActive && (
            <Stack direction="row" gap={1} flexWrap="wrap" alignItems="center">
              <Box sx={{ color: 'text.secondary', fontSize: 13 }}>Filters:</Box>
              {agentGroupParam && (
                <Tooltip title="Open agent group">
                  <Chip
                    icon={<GroupIcon />}
                    label={`Group: ${agentGroupParam}`}
                    onClick={() => router.push(`/agentgroups/${agentGroupParam}`)}
                    onDelete={clearGroup}
                    color="primary"
                    variant="outlined"
                    size="small"
                  />
                </Tooltip>
              )}
              {qParam && (
                <Chip
                  icon={<SearchIcon />}
                  label={`UID contains: ${qParam}`}
                  onDelete={clearSearch}
                  color="primary"
                  variant="outlined"
                  size="small"
                />
              )}
              {descParam && (
                <Chip
                  icon={<SearchIcon />}
                  label={`Description: ${descParam}`}
                  onDelete={() => {
                    setQuery('');
                    updateUrl({ desc: '' });
                  }}
                  color="primary"
                  variant="outlined"
                  size="small"
                />
              )}
            </Stack>
          )}
        </Stack>
      </Paper>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      <Box sx={{ display: 'flex', justifyContent: 'flex-end', mb: 1 }}>
        <ColumnPicker columns={AGENT_COLUMNS} visible={visible} onToggle={toggle} />
      </Box>

      <TableContainer component={Paper}>
        <Table>
          <TableHead>
            <TableRow>
              {isVisible('instanceUid') && <TableCell>Instance UID</TableCell>}
              {isVisible('connected') && <TableCell>Connected</TableCell>}
              {isVisible('healthy') && <TableCell>Healthy</TableCell>}
              {isVisible('type') && <TableCell>Type</TableCell>}
              {isVisible('lastReported') && <TableCell>Last Reported</TableCell>}
              {isVisible('sequence') && <TableCell>Sequence</TableCell>}
              {isVisible('capabilities') && <TableCell>Capabilities</TableCell>}
              {isVisible('identifyingAttributes') && (
                <TableCell>Description (identifying attributes)</TableCell>
              )}
              {isVisible('nonIdentifyingAttributes') && (
                <TableCell>Description (non-identifying attributes)</TableCell>
              )}
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
            ) : visibleAgents.length === 0 ? (
              <TableRow>
                <TableCell colSpan={colSpan} align="center">
                  {descParam
                    ? `No agents on this page match description "${descParam}"`
                    : 'No agents found'}
                </TableCell>
              </TableRow>
            ) : (
              visibleAgents.map((agent) => (
                <TableRow
                  key={agent.metadata.instanceUid}
                  hover
                  component={Link}
                  href={`/agents/${agent.metadata.instanceUid}`}
                  sx={{
                    textDecoration: 'none',
                    cursor: 'pointer',
                    '&:hover': { bgcolor: 'action.hover' },
                  }}
                >
                  {isVisible('instanceUid') && (
                    <TableCell sx={{ fontFamily: 'monospace' }}>
                      {agent.metadata.instanceUid}
                    </TableCell>
                  )}
                  {isVisible('connected') && (
                    <TableCell>
                      <Chip
                        label={agent.status.connected ? 'Connected' : 'Disconnected'}
                        color={agent.status.connected ? 'success' : 'default'}
                        size="small"
                      />
                    </TableCell>
                  )}
                  {isVisible('healthy') && (
                    <TableCell>
                      <Chip
                        label={agent.status.componentHealth?.healthy ? 'Healthy' : 'Unhealthy'}
                        color={agent.status.componentHealth?.healthy ? 'success' : 'warning'}
                        size="small"
                      />
                    </TableCell>
                  )}
                  {isVisible('type') && <TableCell>{agent.status.connectionType || '-'}</TableCell>}
                  {isVisible('lastReported') && (
                    <TableCell>
                      <TimeDisplay value={agent.status.lastReportedAt} />
                    </TableCell>
                  )}
                  {isVisible('sequence') && (
                    <TableCell>{agent.status.sequenceNum ?? '-'}</TableCell>
                  )}
                  {isVisible('capabilities') && (
                    <TableCell>
                      {(() => {
                        const caps = capabilityNames(agent.metadata.capabilities);
                        return caps.length === 0 ? (
                          '-'
                        ) : (
                          <Stack direction="row" gap={0.5} flexWrap="wrap" sx={{ maxWidth: 320 }}>
                            {caps.map((c) => (
                              <Chip key={c} label={c} size="small" variant="outlined" />
                            ))}
                          </Stack>
                        );
                      })()}
                    </TableCell>
                  )}
                  {isVisible('identifyingAttributes') && (
                    <TableCell>
                      <AttrChips attrs={agent.metadata.description?.identifyingAttributes} />
                    </TableCell>
                  )}
                  {isVisible('nonIdentifyingAttributes') && (
                    <TableCell>
                      <AttrChips attrs={agent.metadata.description?.nonIdentifyingAttributes} />
                    </TableCell>
                  )}
                  <TableCell align="right">
                    <RowActionsMenu
                      actions={[
                        {
                          label: 'View detail',
                          icon: <ViewIcon fontSize="small" />,
                          href: `/agents/${agent.metadata.instanceUid}`,
                        },
                        {
                          label: 'Edit spec',
                          icon: <EditIcon fontSize="small" />,
                          href: `/agents/${agent.metadata.instanceUid}?action=edit`,
                        },
                        {
                          label: 'Request restart',
                          icon: <RestartIcon fontSize="small" />,
                          href: `/agents/${agent.metadata.instanceUid}?action=restart`,
                        },
                        // Deleting a connected agent is pointless (it reappears),
                        // so only surface it for disconnected ones.
                        ...(!agent.status.connected
                          ? [
                              {
                                label: 'Delete agent',
                                icon: <DeleteIcon fontSize="small" />,
                                onClick: () => setDeleting(agent),
                                destructive: true,
                                divider: true,
                              } satisfies RowAction,
                            ]
                          : []),
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

      <ConfirmDialog
        open={deleting !== null}
        title="Delete agent"
        message={deleting ? agentDeleteConfirmMessage(deleting.metadata.instanceUid) : ''}
        confirmLabel="Delete"
        destructive
        onClose={() => setDeleting(null)}
        onConfirm={onDelete}
      />
    </Box>
  );
}

export default function AgentsPage() {
  return (
    <Suspense fallback={null}>
      <AgentsInner />
    </Suspense>
  );
}
