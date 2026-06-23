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
import { useEffect, useState } from 'react';
import { api, useApi, type ListResponse } from '@shared/api';
import {
  describeRemoteConfigSources,
  remoteConfigRefs,
  withRemoteConfigRefs,
  type AgentGroup,
} from '@entities/agent-group';
import type { AgentRemoteConfig } from '@entities/agent-remote-config';

interface Props {
  open: boolean;
  namespace: string;
  config: AgentRemoteConfig | null;
  onClose: () => void;
  onApplied: () => void;
}

export default function ApplyToGroupDialog({ open, namespace, config, onClose, onApplied }: Props) {
  const [selected, setSelected] = useState('');
  const [busy, setBusy] = useState(false);
  const [applyError, setApplyError] = useState<string | null>(null);

  // Only fetch while the dialog is open (null key disables the request).
  const {
    data,
    error: fetchError,
    isLoading: loading,
  } = useApi<ListResponse<AgentGroup>>(
    open ? [`/api/v1/namespaces/${namespace}/agentgroups`, { limit: 200 }] : null,
  );
  const groups = data?.items ?? [];
  const error =
    applyError ??
    (fetchError instanceof Error
      ? fetchError.message
      : fetchError
        ? 'Failed to fetch agent groups'
        : null);

  useEffect(() => {
    if (!open) {
      setSelected('');
      setApplyError(null);
    }
  }, [open]);

  const apply = async () => {
    if (!config || !selected) return;
    setBusy(true);
    setApplyError(null);
    try {
      // Re-fetch the group to apply on top of its latest state, then add this
      // resource to the group's remote configs via agentRemoteConfigRef
      // (preserving any configs already applied; duplicates are collapsed).
      const group = await api.get<AgentGroup>(
        `/api/v1/namespaces/${namespace}/agentgroups/${selected}`,
      );
      const nextRefs = [...remoteConfigRefs(group), config.metadata.name];
      const body: AgentGroup = {
        ...group,
        spec: {
          ...group.spec,
          agentConfig: {
            ...group.spec.agentConfig,
            agentRemoteConfigs: withRemoteConfigRefs(group, nextRefs),
          },
        },
      };
      await api.put(`/api/v1/namespaces/${namespace}/agentgroups/${selected}`, body);
      onApplied();
    } catch (err) {
      setApplyError(err instanceof Error ? err.message : 'Failed to apply');
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
            Add <code>{config?.metadata.name}</code> to an agent group&apos;s remote configs. The
            group&apos;s matching agents will receive this config alongside any already applied.
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
              Current remote config: <code>{describeRemoteConfigSources(selectedGroup)}</code> ·{' '}
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
