'use client';

import { useEffect, useState } from 'react';
import {
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
  CircularProgress,
  Box,
  Alert,
  IconButton,
  Chip,
} from '@mui/material';
import { Refresh as RefreshIcon } from '@mui/icons-material';

interface Agent {
  id: string;
  metadata?: {
    instanceUid: string;
    [key: string]: any;
  };
  spec?: {
    [key: string]: any;
  };
  status?: {
    effectiveConfig?: any;
    packageStatuses?: any;
    componentHealth?: any;
    availableComponents?: any;
    conditions?: any[];
    connected?: boolean;
    connectionType?: string;
    lastReportedAt?: string;
  };
  [key: string]: any;
}

export default function AgentsPage() {
  const [agents, setAgents] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchAgents = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await fetch('/api/agents');
      
      if (!response.ok) {
        throw new Error('Failed to fetch agents');
      }
      
      const data = await response.json();
      setAgents(data.items || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAgents();
  }, []);

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return <Alert severity="error">{error}</Alert>;
  }

  return (
    <Box>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Typography variant="h4" component="h1">
          Agents
        </Typography>
        <IconButton onClick={fetchAgents} color="primary">
          <RefreshIcon />
        </IconButton>
      </Box>

      <TableContainer component={Paper}>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>ID</TableCell>
              <TableCell>Instance UID</TableCell>
              <TableCell>Connection</TableCell>
              <TableCell>Connection Type</TableCell>
              <TableCell>Last Reported</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {agents.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} align="center">
                  No agents found
                </TableCell>
              </TableRow>
            ) : (
              agents.map((agent, index) => (
                <TableRow key={agent.id || `agent-${index}`} hover>
                  <TableCell>{agent.id}</TableCell>
                  <TableCell>{agent.metadata?.instanceUid || '-'}</TableCell>
                  <TableCell>
                    <Chip
                      label={agent.status?.connected ? 'Connected' : 'Disconnected'}
                      color={agent.status?.connected ? 'success' : 'default'}
                      size="small"
                    />
                  </TableCell>
                  <TableCell>{agent.status?.connectionType || '-'}</TableCell>
                  <TableCell>{agent.status?.lastReportedAt || '-'}</TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>
    </Box>
  );
}
