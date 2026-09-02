'use client';

import { useEffect, useRef, useState } from 'react';
import { api, useApi, type ListResponse } from '@shared/api';
import {
  Alert,
  Button,
  Checkbox,
  Dialog,
  DialogBody,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Label,
  Spinner,
} from '@shared/ui';
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
      setApplyError(null);
      didSeed.current = false;
      return;
    }
    if (didSeed.current) return;
    didSeed.current = true;
    setRefs(remoteConfigRefs(groupRef.current));
  }, [open]);

  // Toggle options: every fetched config plus any ref the group already points at that is
  // not in the fetched list (e.g. deleted, or beyond the fetch limit), so it stays
  // deselectable rather than silently stuck on.
  const options = Array.from(new Set([...configs.map((c) => c.metadata.name), ...refs]));

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

  const toggle = (name: string) =>
    setRefs((prev) => (prev.includes(name) ? prev.filter((r) => r !== name) : [...prev, name]));

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent size="sm">
        <DialogHeader>
          <DialogTitle>Apply remote configs</DialogTitle>
        </DialogHeader>
        <DialogBody className="space-y-3">
          <p className="text-sm text-muted-foreground">
            Choose the remote configs applied to agent group{' '}
            <code className="font-mono">{group?.metadata.name}</code>. The group&apos;s matching
            agents receive every selected config.
          </p>
          {group && (
            <p className="text-xs text-muted-foreground">
              Current: <code className="font-mono">{describeRemoteConfigSources(group)}</code> ·{' '}
              {group.status.numAgents} agent(s) matched
            </p>
          )}
          {error && <Alert severity="error">{error}</Alert>}
          <div>
            <Label className="mb-1 block">Remote configs</Label>
            <div className="max-h-64 overflow-y-auto rounded-md border border-border">
              {loading ? (
                <div className="p-4">
                  <Spinner className="mx-auto" />
                </div>
              ) : options.length === 0 ? (
                <p className="p-3 text-sm text-muted-foreground">
                  No remote configs in this namespace.
                </p>
              ) : (
                options.map((name) => (
                  <label
                    key={name}
                    className="flex cursor-pointer items-center gap-2 border-b border-border px-3 py-1.5 text-sm last:border-0 hover:bg-accent"
                  >
                    <Checkbox
                      checked={refs.includes(name)}
                      disabled={busy}
                      onCheckedChange={() => toggle(name)}
                    />
                    <span className="truncate">{name}</span>
                  </label>
                ))
              )}
            </div>
            {refs.length === 0 && options.length > 0 && (
              <p className="mt-1 text-xs text-muted-foreground">
                Nothing selected — matching agents will receive no remote config.
              </p>
            )}
          </div>
        </DialogBody>
        <DialogFooter>
          <Button variant="ghost" onClick={onClose} disabled={busy}>
            Cancel
          </Button>
          <Button onClick={() => void save()} disabled={busy}>
            Apply
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
