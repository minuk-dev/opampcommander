'use client';

import {
  Alert,
  Box,
  Button,
  Chip,
  CircularProgress,
  IconButton,
  Paper,
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
} from '@mui/icons-material';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { Suspense, useCallback, useEffect, useState } from 'react';
import PageHeader from '@/components/PageHeader';
import { useNamespace } from '@/components/NamespaceProvider';
import { api } from '@/lib/api-client';
import type { Agent, ListResponse } from '@/lib/types';

const PAGE_SIZE = 50;

function AgentsInner() {
  const router = useRouter();
  const search = useSearchParams();
  const { namespace } = useNamespace();

  const agentGroupParam = search.get('agentGroup') || '';
  const qParam = search.get('q') || '';

  const [agents, setAgents] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [query, setQuery] = useState(qParam);
  const [continueToken, setContinueToken] = useState<string | null>(null);
  const [continueStack, setContinueStack] = useState<string[]>([]);

  // Keep the local input synced if the URL ?q= changes externally
  useEffect(() => {
    setQuery(qParam);
  }, [qParam]);

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

  const updateUrl = (next: { q?: string; agentGroup?: string }) => {
    const params = new URLSearchParams();
    const q = next.q ?? qParam;
    const g = next.agentGroup ?? agentGroupParam;
    if (q) params.set('q', q);
    if (g) params.set('agentGroup', g);
    const qs = params.toString();
    router.replace(qs ? `/agents?${qs}` : '/agents');
  };

  const onSearch = (e: React.FormEvent) => {
    e.preventDefault();
    updateUrl({ q: query.trim() });
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

  const filterActive = Boolean(agentGroupParam || qParam);

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
            <Stack direction="row" gap={1}>
              <TextField
                size="small"
                fullWidth
                placeholder={
                  agentGroupParam
                    ? 'UID search disabled while group filter is active'
                    : 'Search by instance UID…'
                }
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                disabled={Boolean(agentGroupParam)}
                InputProps={{
                  startAdornment: (
                    <SearchIcon sx={{ mr: 1, color: 'text.secondary' }} />
                  ),
                }}
              />
              <Button
                type="submit"
                variant="contained"
                disabled={Boolean(agentGroupParam)}
              >
                Search
              </Button>
            </Stack>
          </form>

          {filterActive && (
            <Stack direction="row" gap={1} flexWrap="wrap" alignItems="center">
              <Box sx={{ color: 'text.secondary', fontSize: 13 }}>Filters:</Box>
              {agentGroupParam && (
                <Tooltip title="Open agent group">
                  <Chip
                    icon={<GroupIcon />}
                    label={`Group: ${agentGroupParam}`}
                    onClick={() =>
                      router.push(`/agentgroups/${agentGroupParam}`)
                    }
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
            </Stack>
          )}
        </Stack>
      </Paper>

      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

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
            </TableRow>
          </TableHead>
          <TableBody>
            {loading ? (
              <TableRow>
                <TableCell colSpan={6} align="center">
                  <CircularProgress size={24} />
                </TableCell>
              </TableRow>
            ) : agents.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} align="center">
                  No agents found
                </TableCell>
              </TableRow>
            ) : (
              agents.map((agent) => (
                <TableRow
                  key={agent.metadata.instanceUid}
                  hover
                  component={Link}
                  href={`/agents/${agent.metadata.instanceUid}`}
                  sx={{ textDecoration: 'none', cursor: 'pointer' }}
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
