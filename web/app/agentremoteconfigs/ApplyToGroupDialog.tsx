'use client';

import {
  Alert,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  FormControl,
  InputLabel,
  MenuItem,
  Select,
  Stack,
  Typography,
} from '@mui/material';
import { useCallback, useEffect, useState } from 'react';
import { api } from '@/lib/api-client';
import type { AgentGroup, AgentRemoteConfig, ListResponse } from '@/lib/types';

interface Props {
  open: boolean;
  namespace: string;
  config: AgentRemoteConfig | null;
  onClose: () => void;
  onApplied: () => void;
}

// Describe how a group currently sources its remote config, so the user can
// see what an apply would overwrite.
function currentRemoteConfig(g: AgentGroup): string {
  const rc = g.spec.agentConfig?.agentRemoteConfig;
  if (!rc) return 'none';
  if (rc.agentRemoteConfigRef) return `ref → ${rc.agentRemoteConfigRef}`;
  if (rc.agentRemoteConfigName) return `inline (${rc.agentRemoteConfigName})`;
  return 'none';
}

export default function ApplyToGroupDialog({ open, namespace, config, onClose, onApplied }: Props) {
  const [groups, setGroups] = useState<AgentGroup[]>([]);
  const [selected, setSelected] = useState('');
  const [loading, setLoading] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchGroups = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await api.get<ListResponse<AgentGroup>>(
        `/api/v1/namespaces/${namespace}/agentgroups`,
        { query: { limit: 200 } },
      );
      setGroups(data.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch agent groups');
    } finally {
      setLoading(false);
    }
  }, [namespace]);

  useEffect(() => {
    if (!open) {
      setSelected('');
      setError(null);
      return;
    }
    void fetchGroups();
  }, [open, fetchGroups]);

  const apply = async () => {
    if (!config || !selected) return;
    setBusy(true);
    setError(null);
    try {
      // Re-fetch the group to apply on top of its latest state, then point its
      // remote config at this resource via agentRemoteConfigRef (clearing any
      // inline config so the reference takes effect unambiguously).
      const group = await api.get<AgentGroup>(
        `/api/v1/namespaces/${namespace}/agentgroups/${selected}`,
      );
      const body: AgentGroup = {
        ...group,
        spec: {
          ...group.spec,
          agentConfig: {
            ...group.spec.agentConfig,
            agentRemoteConfig: { agentRemoteConfigRef: config.metadata.name },
          },
        },
      };
      await api.put(`/api/v1/namespaces/${namespace}/agentgroups/${selected}`, body);
      onApplied();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to apply');
    } finally {
      setBusy(false);
    }
  };

  const selectedGroup = groups.find((g) => g.metadata.name === selected);

  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="sm">
      <DialogTitle>Apply remote config to agent group</DialogTitle>
      <DialogContent>
        <Stack spacing={2} mt={1}>
          <DialogContentText>
            Point an agent group&apos;s remote config at <code>{config?.metadata.name}</code>. The
            group&apos;s matching agents will receive this config. Any existing remote config on the
            group will be replaced.
          </DialogContentText>
          {error && <Alert severity="error">{error}</Alert>}
          <FormControl fullWidth disabled={loading || busy}>
            <InputLabel id="apply-group-label">Agent group</InputLabel>
            <Select
              labelId="apply-group-label"
              label="Agent group"
              value={selected}
              onChange={(e) => setSelected(e.target.value)}
            >
              {groups.length === 0 && (
                <MenuItem value="" disabled>
                  {loading ? 'Loading…' : 'No agent groups in this namespace'}
                </MenuItem>
              )}
              {groups.map((g) => (
                <MenuItem key={g.metadata.name} value={g.metadata.name}>
                  {g.metadata.name}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
          {selectedGroup && (
            <Typography variant="body2" color="text.secondary">
              Current remote config: <code>{currentRemoteConfig(selectedGroup)}</code> ·{' '}
              {selectedGroup.status.numAgents} agent(s) matched
            </Typography>
          )}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={busy}>
          Cancel
        </Button>
        <Button variant="contained" onClick={apply} disabled={busy || !selected}>
          Apply
        </Button>
      </DialogActions>
    </Dialog>
  );
}
