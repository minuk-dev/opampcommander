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
  Button,
  Chip,
} from '@mui/material';
import { Refresh as RefreshIcon, Add as AddIcon } from '@mui/icons-material';

interface AgentGroup {
  id: string;
  name: string;
  description?: string;
  agentCount?: number;
  createdAt?: string;
  [key: string]: any;
}

export default function AgentGroupsPage() {
  const [agentGroups, setAgentGroups] = useState<AgentGroup[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchAgentGroups = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await fetch('/api/agentgroups');
      
      if (!response.ok) {
        throw new Error('Failed to fetch agent groups');
      }
      
      const data = await response.json();
      setAgentGroups(data.items || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAgentGroups();
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
          Agent Groups
        </Typography>
        <Box>
          <IconButton onClick={fetchAgentGroups} color="primary">
            <RefreshIcon />
          </IconButton>
          <Button variant="contained" startIcon={<AddIcon />} sx={{ ml: 1 }}>
            Create Group
          </Button>
        </Box>
      </Box>

      <TableContainer component={Paper}>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>ID</TableCell>
              <TableCell>Name</TableCell>
              <TableCell>Description</TableCell>
              <TableCell>Agent Count</TableCell>
              <TableCell>Created At</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {agentGroups.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} align="center">
                  No agent groups found
                </TableCell>
              </TableRow>
            ) : (
              agentGroups.map((group) => (
                <TableRow key={group.id} hover>
                  <TableCell>{group.id}</TableCell>
                  <TableCell>{group.name || '-'}</TableCell>
                  <TableCell>{group.description || '-'}</TableCell>
                  <TableCell>
                    <Chip label={group.agentCount || 0} color="primary" size="small" />
                  </TableCell>
                  <TableCell>{group.createdAt || '-'}</TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>
    </Box>
  );
}
