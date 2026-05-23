'use client';

import {
  Alert,
  Autocomplete,
  Box,
  Button,
  Chip,
  CircularProgress,
  FormControl,
  IconButton,
  InputLabel,
  MenuItem,
  Paper,
  Select,
  Stack,
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
} from '@mui/icons-material';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { Suspense, useCallback, useEffect, useState } from 'react';
import PageHeader from '@/components/PageHeader';
import RowActionsMenu from '@/components/RowActionsMenu';
import { useNamespace } from '@/components/NamespaceProvider';
import { api } from '@/lib/api-client';
import type { Agent, AgentGroup, ListResponse } from '@/lib/types';

const PAGE_SIZE = 50;

type SearchMode = 'uid' | 'group' | 'description';

function attrMatchesDescription(agent: Agent, needle: string): boolean {
  const lc = needle.toLowerCase();
  const desc = agent.metadata.description;
  const collect = [
    ...Object.entries(desc?.identifyingAttributes ?? {}),
    ...Object.entries(desc?.nonIdentifyingAttributes ?? {}),
  ];
  return collect.some(([k, v]) => k.toLowerCase().includes(lc) || v.toLowerCase().includes(lc));
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

  const [agents, setAgents] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [mode, setMode] = useState<SearchMode>(modeParam);
  const [groupOptions, setGroupOptions] = useState<AgentGroup[]>([]);
  const [query, setQuery] = useState(qParam);
  const [continueToken, setContinueToken] = useState<string | null>(null);
  const [continueStack, setContinueStack] = useState<string[]>([]);

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

  // Load agent groups for the "Group" autocomplete and the chip cross-reference.
  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const res = await api.get<ListResponse<AgentGroup>>(
          `/api/v1/namespaces/${namespace}/agentgroups`,
          { query: { limit: 500 } },
        );
        if (!cancelled) setGroupOptions(res.items ?? []);
      } catch {
        /* listing might be RBAC-denied; group filter still works via URL */
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [namespace]);

  const fetchAgents = useCallback(
    async (token?: string) => {
      setLoading(true);
      setError(null);
      try {
        let path: string;
        const q: Record<string, string | number | undefined> = {
          limit: PAGE_SIZE,
          continue: token,
        };

        if (agentGroupParam) {
          path = `/api/v1/namespaces/${namespace}/agentgroups/${agentGroupParam}/agents`;
        } else if (qParam) {
          path = `/api/v1/namespaces/${namespace}/agents/search`;
          q.q = qParam;
        } else {
          // Description search filters client-side over the unsearched list.
          path = `/api/v1/namespaces/${namespace}/agents`;
        }

        const data = await api.get<ListResponse<Agent>>(path, { query: q });
        setAgents(data.items ?? []);
        setContinueToken(data.metadata?.continue || null);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch agents');
      } finally {
        setLoading(false);
      }
    },
    [namespace, qParam, agentGroupParam],
  );

  useEffect(() => {
    setContinueStack([]);
    void fetchAgents();
  }, [fetchAgents]);

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

  const next = () => {
    if (!continueToken) return;
    setContinueStack((s) => [...s, continueToken]);
    void fetchAgents(continueToken);
  };

  const filterActive = Boolean(agentGroupParam || qParam || descParam);
  const visibleAgents = descParam
    ? agents.filter((a) => attrMatchesDescription(a, descParam))
    : agents;

  return (
    <Box>
      <PageHeader
        title="Agents"
        subtitle={`Namespace: ${namespace}`}
        actions={
          <IconButton color="primary" onClick={() => fetchAgents()}>
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

      <TableContainer component={Paper}>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>Instance UID</TableCell>
              <TableCell>Connected</TableCell>
              <TableCell>Healthy</TableCell>
              <TableCell>Type</TableCell>
              <TableCell>Last Reported</TableCell>
              <TableCell>Sequence</TableCell>
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
            ) : visibleAgents.length === 0 ? (
              <TableRow>
                <TableCell colSpan={7} align="center">
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
                  <TableCell sx={{ fontFamily: 'monospace' }}>
                    {agent.metadata.instanceUid}
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={agent.status.connected ? 'Connected' : 'Disconnected'}
                      color={agent.status.connected ? 'success' : 'default'}
                      size="small"
                    />
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={agent.status.componentHealth?.healthy ? 'Healthy' : 'Unhealthy'}
                      color={agent.status.componentHealth?.healthy ? 'success' : 'warning'}
                      size="small"
                    />
                  </TableCell>
                  <TableCell>{agent.status.connectionType || '-'}</TableCell>
                  <TableCell>{agent.status.lastReportedAt || '-'}</TableCell>
                  <TableCell>{agent.status.sequenceNum ?? '-'}</TableCell>
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
                      ]}
                    />
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>

      <Stack direction="row" justifyContent="flex-end" mt={2} gap={1}>
        <Button
          disabled={continueStack.length === 0 || loading}
          onClick={() => {
            const prev = continueStack[continueStack.length - 2] || undefined;
            setContinueStack((s) => s.slice(0, -1));
            void fetchAgents(prev);
          }}
        >
          Prev
        </Button>
        <Button disabled={!continueToken || loading} onClick={next}>
          Next
        </Button>
      </Stack>
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
