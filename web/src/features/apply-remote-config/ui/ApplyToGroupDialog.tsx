'use client';

import { useState } from 'react';
import { api, useApi, type ListResponse } from '@shared/api';
import {
  Alert,
  Button,
  Dialog,
  DialogBody,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Field,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
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
  config: AgentRemoteConfig | null;
  onClose: () => void;
  onApplied: () => void;
}

export default function ApplyToGroupDialog({
  open,
  namespace: currentNamespace,
  config: incoming,
  onClose,
  onApplied,
}: Props) {
  const [config, setConfig] = useState(incoming);
  const [namespace, setNamespace] = useState(incoming?.metadata.namespace ?? currentNamespace);
  const [selectedGroup, setSelectedGroup] = useState<AgentGroup | null>(null);
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

  // Pin the source and namespace for this open; list revalidation must not
  // replace the group revision the user selected either.
  const [wasOpen, setWasOpen] = useState(false);
  if (open !== wasOpen) {
    setWasOpen(open);
    setSelectedGroup(null);
    setApplyError(null);
    if (open) {
      setConfig(incoming);
      setNamespace(incoming?.metadata.namespace ?? currentNamespace);
    }
  }

  const apply = async () => {
    if (!config || !selectedGroup) return;
    setBusy(true);
    setApplyError(null);
    try {
      const group = selectedGroup;
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
      await api.put(
        `/api/v1/namespaces/${group.metadata.namespace}/agentgroups/${group.metadata.name}`,
        body,
      );
      onApplied();
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to apply';
      setApplyError(
        err instanceof Error && 'status' in err && err.status === 409
          ? `${message} Close this dialog, refresh the list, and reopen it before applying again.`
          : message,
      );
    } finally {
      setBusy(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent size="sm">
        <DialogHeader>
          <DialogTitle>Apply remote config to agent group</DialogTitle>
        </DialogHeader>
        <DialogBody className="space-y-3">
          <p className="text-sm text-muted-foreground">
            Add <code className="font-mono">{config?.metadata.name}</code> to an agent group&apos;s
            remote configs. The group&apos;s matching agents will receive this config alongside any
            already applied.
          </p>
          {error && <Alert severity="error">{error}</Alert>}
          <Field
            label="Agent group"
            hint={
              selectedGroup
                ? `Current remote config: ${describeRemoteConfigSources(selectedGroup)} · ${selectedGroup.status.numAgents} agent(s) matched`
                : undefined
            }
          >
            {(field) => (
              <Select
                value={selectedGroup?.metadata.name ?? ''}
                onValueChange={(name) =>
                  setSelectedGroup(groups.find((group) => group.metadata.name === name) ?? null)
                }
                disabled={loading || busy || groups.length === 0}
              >
                <SelectTrigger {...field}>
                  <SelectValue
                    placeholder={loading ? 'Loading…' : 'No agent groups in this namespace'}
                  >
                    {selectedGroup?.metadata.name}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {groups.map((g) => (
                    <SelectItem key={g.metadata.name} value={g.metadata.name}>
                      {g.metadata.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
          </Field>
        </DialogBody>
        <DialogFooter>
          <Button variant="ghost" onClick={onClose} disabled={busy}>
            Cancel
          </Button>
          <Button onClick={() => void apply()} disabled={busy || !selectedGroup || !config}>
            Apply
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
