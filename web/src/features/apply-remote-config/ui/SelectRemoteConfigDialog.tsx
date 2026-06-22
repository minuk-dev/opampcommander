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
import { useEffect, useRef, useState } from 'react';
import { api, useApi, type ListResponse } from '@shared/api';
import { ConfirmDialog } from '@shared/ui';
import {
  currentRemoteConfigRef,
  describeRemoteConfigSource,
  hasRemoteConfig,
  type AgentGroup,
} from '@entities/agent-group';
import type { AgentRemoteConfig } from '@entities/agent-remote-config';

interface Props {
  open: boolean;
  namespace: string;
  group: AgentGroup | null;
  onClose: () => void;
  onApplied: () => void;
}

export default function SelectRemoteConfigDialog({
  open,
  namespace,
  group,
  onClose,
  onApplied,
}: Props) {
  const [selected, setSelected] = useState('');
  const [busy, setBusy] = useState(false);
  const [confirmingRemove, setConfirmingRemove] = useState(false);
  const [applyError, setApplyError] = useState<string | null>(null);

  // Only fetch while the dialog is open (null key disables the request).
  const {
    data,
    error: fetchError,
    isLoading: loading,
  } = useApi<ListResponse<AgentRemoteConfig>>(
    open ? [`/api/v1/namespaces/${namespace}/agentremoteconfigs`, { limit: 200 }] : null,
  );
  const configs = data?.items ?? [];
  const error =
    applyError ??
    (fetchError instanceof Error
      ? fetchError.message
      : fetchError
        ? 'Failed to fetch remote configs'
        : null);

  // Read the latest group through a ref so the preselect effect can key on
  // `open` alone: re-running it on every `group` change would clobber an
  // in-progress selection whenever SWR revalidates the group in the background.
  const groupRef = useRef(group);
  groupRef.current = group;

  // Preselect the group's current ref (if any) only when the dialog opens.
  useEffect(() => {
    if (!open) {
      setSelected('');
      setApplyError(null);
      return;
    }
    setSelected(currentRemoteConfigRef(groupRef.current));
  }, [open]);

  // Write the group's remote config: a ref points it at the named resource,
  // null clears any existing remote config off the group.
  const save = async (ref: string | null) => {
    if (!group) return;
    setBusy(true);
    setApplyError(null);
    try {
      // Re-fetch the group to apply on top of its latest state. Setting a ref
      // clears any inline config so the reference takes effect unambiguously;
      // clearing drops the agentRemoteConfig entirely.
      const latest = await api.get<AgentGroup>(
        `/api/v1/namespaces/${namespace}/agentgroups/${group.metadata.name}`,
      );
      const body: AgentGroup = {
        ...latest,
        spec: {
          ...latest.spec,
          agentConfig: {
            ...latest.spec.agentConfig,
            agentRemoteConfig: ref ? { agentRemoteConfigRef: ref } : undefined,
          },
        },
      };
      await api.put(`/api/v1/namespaces/${namespace}/agentgroups/${group.metadata.name}`, body);
      onApplied();
    } catch (err) {
      setApplyError(err instanceof Error ? err.message : 'Failed to apply');
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <Dialog open={open} onClose={onClose} fullWidth maxWidth="sm">
        <DialogTitle>Apply remote config</DialogTitle>
        <DialogContent>
          <Stack spacing={2} mt={1}>
            <DialogContentText>
              Point agent group <code>{group?.metadata.name}</code> at a remote config. The
              group&apos;s matching agents will receive this config. Any existing remote config on
              the group will be replaced, or use Remove to clear it.
            </DialogContentText>
            {group && (
              <Typography variant="body2" color="text.secondary">
                Current remote config: <code>{describeRemoteConfigSource(group)}</code> ·{' '}
                {group.status.numAgents} agent(s) matched
              </Typography>
            )}
            {error && <Alert severity="error">{error}</Alert>}
            <FormControl fullWidth disabled={loading || busy}>
              <InputLabel id="select-remote-config-label">Remote config</InputLabel>
              <Select
                labelId="select-remote-config-label"
                label="Remote config"
                value={selected}
                onChange={(e) => setSelected(e.target.value)}
              >
                {configs.length === 0 && (
                  <MenuItem value="" disabled>
                    {loading ? 'Loading…' : 'No remote configs in this namespace'}
                  </MenuItem>
                )}
                {configs.map((c) => (
                  <MenuItem key={c.metadata.name} value={c.metadata.name}>
                    {c.metadata.name}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          </Stack>
        </DialogContent>
        <DialogActions>
          {hasRemoteConfig(group) && (
            <Button
              color="error"
              onClick={() => setConfirmingRemove(true)}
              disabled={busy}
              sx={{ mr: 'auto' }}
            >
              Remove
            </Button>
          )}
          <Button onClick={onClose} disabled={busy}>
            Cancel
          </Button>
          <Button variant="contained" onClick={() => save(selected)} disabled={busy || !selected}>
            Apply
          </Button>
        </DialogActions>
      </Dialog>
      <ConfirmDialog
        open={confirmingRemove}
        title="Remove remote config"
        message={`Clear the remote config from "${group?.metadata.name}"? Matching agents will stop receiving it.`}
        confirmLabel="Remove"
        destructive
        onClose={() => setConfirmingRemove(false)}
        onConfirm={async () => {
          setConfirmingRemove(false);
          await save(null);
        }}
      />
    </>
  );
}
