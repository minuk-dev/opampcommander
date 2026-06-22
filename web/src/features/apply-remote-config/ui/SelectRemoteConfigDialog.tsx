'use client';

import {
  Alert,
  Box,
  Button,
  Chip,
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
  // The working set of refs the user is editing; applied to the group on Apply.
  const [refs, setRefs] = useState<string[]>([]);
  const [toAdd, setToAdd] = useState('');
  const [busy, setBusy] = useState(false);
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

  // Read the latest group through a ref so the seed effect doesn't have to
  // depend on `group`: re-running it on every `group` change would clobber an
  // in-progress edit whenever SWR revalidates the group in the background.
  const groupRef = useRef(group);
  groupRef.current = group;

  // Seed the working set from the group's current refs once per open.
  // `didSeed` keeps a later SWR revalidation from clobbering the user's edits.
  const didSeed = useRef(false);
  useEffect(() => {
    if (!open) {
      setRefs([]);
      setToAdd('');
      setApplyError(null);
      didSeed.current = false;
      return;
    }
    if (didSeed.current) return;
    didSeed.current = true;
    setRefs(remoteConfigRefs(groupRef.current));
  }, [open]);

  // Configs available to add: those not already in the working set.
  const available = configs.filter((c) => !refs.includes(c.metadata.name));

  const addRef = () => {
    if (!toAdd || refs.includes(toAdd)) return;
    setRefs((prev) => [...prev, toAdd]);
    setToAdd('');
  };

  const removeRef = (ref: string) => {
    setRefs((prev) => prev.filter((r) => r !== ref));
  };

  // Persist the working set onto the group, preserving any inline (non-ref)
  // entries that were defined outside this dialog.
  const save = async () => {
    if (!group) return;
    setBusy(true);
    setApplyError(null);
    try {
      // Re-fetch the group to apply on top of its latest state.
      const latest = await api.get<AgentGroup>(
        `/api/v1/namespaces/${namespace}/agentgroups/${group.metadata.name}`,
      );
      const nextConfigs = withRemoteConfigRefs(latest, refs);
      const body: AgentGroup = {
        ...latest,
        spec: {
          ...latest.spec,
          agentConfig: {
            ...latest.spec.agentConfig,
            agentRemoteConfigs: nextConfigs.length > 0 ? nextConfigs : undefined,
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
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="sm">
      <DialogTitle>Apply remote configs</DialogTitle>
      <DialogContent>
        <Stack spacing={2} mt={1}>
          <DialogContentText>
            Choose the remote configs applied to agent group <code>{group?.metadata.name}</code>.
            The group&apos;s matching agents receive every config in the list. Add or remove configs,
            then Apply.
          </DialogContentText>
          {group && (
            <Typography variant="body2" color="text.secondary">
              Current: <code>{describeRemoteConfigSources(group)}</code> · {group.status.numAgents}{' '}
              agent(s) matched
            </Typography>
          )}
          {error && <Alert severity="error">{error}</Alert>}
          {refs.length > 0 ? (
            <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
              {refs.map((ref) => (
                <Chip key={ref} label={ref} onDelete={() => removeRef(ref)} disabled={busy} />
              ))}
            </Box>
          ) : (
            <Typography variant="body2" color="text.secondary">
              No remote configs selected. Matching agents will receive none.
            </Typography>
          )}
          <Stack direction="row" spacing={1} alignItems="flex-start">
            <FormControl fullWidth disabled={loading || busy}>
              <InputLabel id="select-remote-config-label">Add remote config</InputLabel>
              <Select
                labelId="select-remote-config-label"
                label="Add remote config"
                value={toAdd}
                onChange={(e) => setToAdd(e.target.value)}
              >
                {available.length === 0 && (
                  <MenuItem value="" disabled>
                    {loading
                      ? 'Loading…'
                      : configs.length === 0
                        ? 'No remote configs in this namespace'
                        : 'All configs already added'}
                  </MenuItem>
                )}
                {available.map((c) => (
                  <MenuItem key={c.metadata.name} value={c.metadata.name}>
                    {c.metadata.name}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
            <Button onClick={addRef} disabled={busy || !toAdd} sx={{ mt: 1 }}>
              Add
            </Button>
          </Stack>
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={busy}>
          Cancel
        </Button>
        <Button variant="contained" onClick={save} disabled={busy}>
          Apply
        </Button>
      </DialogActions>
    </Dialog>
  );
}
