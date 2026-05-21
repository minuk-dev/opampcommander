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
} from '@mui/material';
import { Refresh as RefreshIcon, Search as SearchIcon } from '@mui/icons-material';
import Link from 'next/link';
import { useCallback, useEffect, useState } from 'react';
import PageHeader from '@/components/PageHeader';
import { useNamespace } from '@/components/NamespaceProvider';
import { api } from '@/lib/api-client';
import type { Agent, ListResponse } from '@/lib/types';

const PAGE_SIZE = 50;

export default function AgentsPage() {
  const { namespace } = useNamespace();
  const [agents, setAgents] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [query, setQuery] = useState('');
  const [submittedQuery, setSubmittedQuery] = useState('');
  const [continueToken, setContinueToken] = useState<string | null>(null);
  const [continueStack, setContinueStack] = useState<string[]>([]);

  const fetchAgents = useCallback(
    async (token?: string) => {
      setLoading(true);
      setError(null);
      try {
        const path = submittedQuery
          ? `/api/v1/namespaces/${namespace}/agents/search`
          : `/api/v1/namespaces/${namespace}/agents`;
        const data = await api.get<ListResponse<Agent>>(path, {
          query: {
            limit: PAGE_SIZE,
            continue: token,
            q: submittedQuery || undefined,
          },
        });
        setAgents(data.items ?? []);
        setContinueToken(data.metadata?.continue || null);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch agents');
      } finally {
        setLoading(false);
      }
    },
    [namespace, submittedQuery],
  );

  useEffect(() => {
    setContinueStack([]);
    void fetchAgents();
  }, [fetchAgents]);

  const onSearch = (e: React.FormEvent) => {
    e.preventDefault();
    setSubmittedQuery(query.trim());
  };

  const next = () => {
    if (!continueToken) return;
    setContinueStack((s) => [...s, continueToken]);
    void fetchAgents(continueToken);
  };

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
        <form onSubmit={onSearch}>
          <Stack direction="row" gap={1}>
            <TextField
              size="small"
              fullWidth
              placeholder="Search by instance UID…"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              InputProps={{
                startAdornment: <SearchIcon sx={{ mr: 1, color: 'text.secondary' }} />,
              }}
            />
            <Button type="submit" variant="contained">
              Search
            </Button>
            {submittedQuery && (
              <Button
                variant="text"
                onClick={() => {
                  setQuery('');
                  setSubmittedQuery('');
                }}
              >
                Clear
              </Button>
            )}
          </Stack>
        </form>
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
